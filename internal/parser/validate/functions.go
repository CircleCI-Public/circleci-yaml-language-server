package validate

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/paramref"
)

// These follow the compiler's config_compilation/functions.clj, and its
// messages are reused so that the editor and the pipeline say the same thing.
var (
	functionIDPattern               = regexp.MustCompile(`^[a-z][a-z\d_-]*$`)
	functionReferencePathPattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*\.[a-z]{2,}(/[A-Za-z0-9._-]+){2,}$`)
	functionReferenceVersionPattern = regexp.MustCompile(`^v\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`)
)

const functionReferenceExample = "github.com/circleci-functions/setup-go@v0.1.0-abc1234"

// Steps a function alias can't share a name with. The macro steps are
// included, as the compiler includes them.
var functionReservedStepNames = []string{
	"run", "checkout", "setup_remote_docker", "save_cache", "restore_cache",
	"deploy", "store_artifacts", "store_test_results", "persist_to_workspace",
	"attach_workspace", "add_ssh_keys", "install_signing_bundle",
	"when", "unless", "steps", "with_tool_cache",
}

// functionStep is a step that runs a declared function.
type functionStep struct {
	step     ast.NamedStep
	function ast.Function
	// command is what the step names after a `/`, if anything.
	command string
	// context names the job or command the step is in, for messages.
	context string
}

func (step functionStep) where() string {
	return fmt.Sprintf("function step '%s' in %s", step.step.Name, step.context)
}

// ValidateFunctions checks the steps that run a declared function, and the
// declarations they use. A declaration nothing uses isn't checked, as the
// compiler doesn't check it.
func (val Validate) ValidateFunctions() {
	if len(val.Doc.Functions) == 0 {
		return
	}

	val.validateFunctionAliases()

	var steps []functionStep
	for _, job := range val.Doc.Jobs {
		steps = append(steps, val.functionSteps(job.Steps, fmt.Sprintf("job '%s'", job.Name))...)
	}
	for _, command := range val.Doc.Commands {
		steps = append(steps, val.functionSteps(command.Steps, fmt.Sprintf("command '%s'", command.Name))...)
	}

	ids := map[string]map[string]int{}
	for _, step := range steps {
		val.validateFunctionCommand(step.step, step.where(), step.command)
		if id, ok := val.validateFunctionStepBody(step.step, step.where()); ok {
			if ids[step.context] == nil {
				ids[step.context] = map[string]int{}
			}
			ids[step.context][id]++
		}
	}
	val.validateUniqueFunctionIDs(steps, ids)

	descriptors := map[string]*circleci.FunctionDescriptor{}
	for _, step := range steps {
		alias := step.function.Alias
		if _, checked := descriptors[alias]; checked {
			continue
		}

		descriptors[alias] = nil
		if val.validateFunctionReference(step.function) {
			descriptors[alias] = val.validateFunctionPublished(step.function)
		}
	}

	for _, step := range steps {
		if descriptor := descriptors[step.function.Alias]; descriptor != nil {
			val.validateFunctionStepAgainst(step, descriptor)
		}
	}
}

func (val Validate) functionSteps(steps []ast.Step, context string) []functionStep {
	var found []functionStep
	for _, step := range steps {
		namedStep, ok := step.(ast.NamedStep)
		if !ok {
			continue
		}

		if function, command, ok := val.Doc.FunctionForStep(namedStep.Name); ok {
			found = append(found, functionStep{step: namedStep, function: function, command: command, context: context})
		}
	}
	return found
}

func (val Validate) validateFunctionAliases() {
	for _, function := range val.Doc.Functions {
		var clashes []string
		if _, ok := val.Doc.Orbs[function.Alias]; ok {
			clashes = append(clashes, "an orb")
		}
		if _, ok := val.Doc.Commands[function.Alias]; ok {
			clashes = append(clashes, "a command")
		}
		if slices.Contains(functionReservedStepNames, function.Alias) {
			clashes = append(clashes, "a built-in step")
		}

		for _, clash := range clashes {
			val.addDiagnostic(diagnostic.Error(
				function.AliasRange,
				fmt.Sprintf("Function name '%s' is also %s. Rename one of them.", function.Alias, clash),
			))
		}
	}
}

func (val Validate) validateUniqueFunctionIDs(steps []functionStep, ids map[string]map[string]int) {
	for _, step := range steps {
		id, ok := step.step.Parameters["id"]
		if !ok {
			continue
		}

		value := fmt.Sprint(id.Value)
		if ids[step.context][value] < 2 {
			continue
		}

		var duplicates []string
		for other, count := range ids[step.context] {
			if count > 1 {
				duplicates = append(duplicates, other)
			}
		}
		sort.Strings(duplicates)

		val.addDiagnostic(diagnostic.Error(
			id.Range,
			fmt.Sprintf("The %s has more than one function step with id: %s", step.context, strings.Join(duplicates, ", ")),
		))
	}
}

func (val Validate) validateFunctionCommand(step ast.NamedStep, where, command string) {
	switch {
	case strings.HasSuffix(step.Name, "/") && command == "":
		val.addDiagnostic(diagnostic.Error(
			step.Range,
			fmt.Sprintf("The %s has a trailing '/' with no command name.", where),
		))
	case strings.Contains(command, "/"):
		val.addDiagnostic(diagnostic.Error(
			step.Range,
			fmt.Sprintf("The %s names more than one command. A step runs the function or one of its commands, and nothing deeper.", where),
		))
	}
}

// validateFunctionStepBody checks the step's body, and returns its id if it
// has a valid one.
func (val Validate) validateFunctionStepBody(step ast.NamedStep, where string) (string, bool) {
	var unexpected []string
	for key := range step.Parameters {
		if key != "id" && key != "with" {
			unexpected = append(unexpected, key)
		}
	}
	if len(unexpected) > 0 {
		sort.Strings(unexpected)
		val.addDiagnostic(diagnostic.Error(
			step.Range,
			fmt.Sprintf("Unexpected key(s) in %s: %s. Pass arguments under `with`.", where, strings.Join(unexpected, ", ")),
		))
	}

	if with, ok := step.Parameters["with"]; ok {
		val.validateFunctionArguments(with, where)
	}

	id, ok := step.Parameters["id"]
	if !ok {
		return "", false
	}

	value, isString := id.Value.(string)
	if id.Type != "string" || !isString || !functionIDPattern.MatchString(value) {
		val.addDiagnostic(diagnostic.Error(
			id.Range,
			fmt.Sprintf("The %s has an invalid id: %s", where, formatFunctionValue(id)),
		))
		return "", false
	}

	return value, true
}

func (val Validate) validateFunctionArguments(with ast.ParameterValue, where string) {
	switch with.Type {
	case "map":
	case "":
		// A flow mapping, which the parser doesn't read.
		return
	default:
		val.addDiagnostic(diagnostic.Error(
			with.Range,
			fmt.Sprintf("`with` in the %s must be a map.", where),
		))
		return
	}

	arguments, _ := with.Value.(map[string]ast.ParameterValue)

	// Arguments are passed to a binary, so each has to be a single value.
	var nested []string
	for name, argument := range arguments {
		switch argument.Type {
		case "map", "enum", "steps":
			nested = append(nested, name)
		}
	}
	if len(nested) == 0 {
		return
	}

	sort.Strings(nested)
	val.addDiagnostic(diagnostic.Error(
		with.Range,
		fmt.Sprintf("Argument(s) in the %s must be a single value, not a list or map: %s", where, strings.Join(nested, ", ")),
	))
}

// validateFunctionReference checks the form of a declaration, and reports
// whether it is well formed.
func (val Validate) validateFunctionReference(function ast.Function) bool {
	rng := function.ReferenceRange
	if !function.IsString {
		rng = function.Range
	}

	report := func(detail string) {
		val.addDiagnostic(diagnostic.Error(rng, fmt.Sprintf("Function '%s' %s", function.Alias, detail)))
	}

	if !function.IsString {
		report(fmt.Sprintf("must be a string, for example %s", functionReferenceExample))
		return false
	}

	path, version, found := strings.Cut(function.Reference, "@")
	if !found {
		report(fmt.Sprintf("is missing an '@version' suffix, which pins the release: %s", functionReferenceExample))
		return false
	}

	if !functionReferencePathPattern.MatchString(path) {
		report(fmt.Sprintf("has an invalid path: %q. Expected host.tld/org/name, as in %s", path, functionReferenceExample))
		return false
	}

	if !functionReferenceVersionPattern.MatchString(version) {
		report(fmt.Sprintf("has an invalid version: %q. Expected a semver tag led by 'v', as in v0.1.0 or v0.1.0-abc1234", version))
		return false
	}

	return true
}

// The catalog is checked with warnings: the compiler doesn't consult it, and
// a step that disagrees with it fails only when the job runs.

// validateFunctionPublished checks a declaration against the functions
// catalog, and returns the descriptor of the version it pins, if it is found.
func (val Validate) validateFunctionPublished(function ast.Function) *circleci.FunctionDescriptor {
	published, descriptor, err := val.Doc.LookUpFunction(function, val.Cache)
	if err != nil {
		return nil
	}

	path, version, _ := strings.Cut(function.Reference, "@")
	switch {
	case published == nil:
		val.addDiagnostic(diagnostic.Warning(
			function.ReferenceRange,
			fmt.Sprintf("No function %s is published.", path),
		))
	case descriptor == nil:
		val.addDiagnostic(diagnostic.Warning(
			function.ReferenceRange,
			fmt.Sprintf("Function %s has no version %s. The latest is %s.", path, version, published.LatestVersion),
		))
	}

	return descriptor
}

// validateFunctionStepAgainst checks a step against the descriptor of the
// function version it runs: the command it names, and the flags it passes
// under `with`.
func (val Validate) validateFunctionStepAgainst(step functionStep, descriptor *circleci.FunctionDescriptor) {
	flags := descriptor.Flags
	if step.command != "" && !strings.Contains(step.command, "/") {
		command, ok := descriptor.Commands[step.command]
		if !ok {
			val.addDiagnostic(diagnostic.Warning(
				step.step.Range,
				fmt.Sprintf("Function %s %s has no command %s.", step.function.Alias, descriptor.Version, step.command),
			))
			return
		}
		flags = command.Flags
	}

	with, ok := step.step.Parameters["with"]
	if !ok || with.Type != "map" {
		return
	}
	arguments, _ := with.Value.(map[string]ast.ParameterValue)

	for name, argument := range arguments {
		index := slices.IndexFunc(flags, func(flag circleci.FunctionFlag) bool { return flag.Name == name })
		if index == -1 {
			val.addDiagnostic(diagnostic.Warning(
				argument.Range,
				fmt.Sprintf("%s %s takes no flag %s.", step.step.Name, descriptor.Version, name),
			))
			continue
		}

		if flags[index].Type == "bool" && !isBooleanArgument(argument) {
			val.addDiagnostic(diagnostic.Warning(
				argument.Range,
				fmt.Sprintf("Flag %s of %s takes true or false.", name, step.step.Name),
			))
		}
	}
}

func isBooleanArgument(argument ast.ParameterValue) bool {
	switch argument.Type {
	case "boolean":
		return true
	case "string":
		value := fmt.Sprint(argument.Value)
		isReference, _ := paramref.IsPartiallyReferenced(value)
		return isReference || value == "true" || value == "false"
	}
	return false
}

func formatFunctionValue(value ast.ParameterValue) string {
	if value.Type == "string" {
		return fmt.Sprintf("%q", fmt.Sprint(value.Value))
	}
	return fmt.Sprint(value.Value)
}
