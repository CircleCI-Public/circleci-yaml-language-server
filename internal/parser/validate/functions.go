package validate

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
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

// ValidateFunctions checks the steps that run a declared function, and the
// declarations they use. A declaration nothing uses isn't checked, as the
// compiler doesn't check it.
func (val Validate) ValidateFunctions() {
	if len(val.Doc.Functions) == 0 {
		return
	}

	val.validateFunctionAliases()

	used := map[string]bool{}
	for _, job := range val.Doc.Jobs {
		val.validateFunctionSteps(job.Steps, fmt.Sprintf("job '%s'", job.Name), used)
	}
	for _, command := range val.Doc.Commands {
		val.validateFunctionSteps(command.Steps, fmt.Sprintf("command '%s'", command.Name), used)
	}

	for alias := range used {
		val.validateFunctionReference(val.Doc.Functions[alias])
	}
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

func (val Validate) validateFunctionSteps(steps []ast.Step, context string, used map[string]bool) {
	ids := map[string]int{}
	for _, step := range steps {
		namedStep, ok := step.(ast.NamedStep)
		if !ok {
			continue
		}

		function, command, ok := val.Doc.FunctionForStep(namedStep.Name)
		if !ok {
			continue
		}
		used[function.Alias] = true

		where := fmt.Sprintf("function step '%s' in %s", namedStep.Name, context)
		val.validateFunctionCommand(namedStep, where, command)
		if id, ok := val.validateFunctionStepBody(namedStep, where); ok {
			ids[id]++
		}
	}

	var duplicates []string
	for id, count := range ids {
		if count > 1 {
			duplicates = append(duplicates, id)
		}
	}
	if len(duplicates) == 0 {
		return
	}
	sort.Strings(duplicates)

	for _, step := range steps {
		namedStep, ok := step.(ast.NamedStep)
		if !ok {
			continue
		}
		if _, _, ok := val.Doc.FunctionForStep(namedStep.Name); !ok {
			continue
		}
		if id, ok := namedStep.Parameters["id"]; ok && slices.Contains(duplicates, fmt.Sprint(id.Value)) {
			val.addDiagnostic(diagnostic.Error(
				id.Range,
				fmt.Sprintf("The %s has more than one function step with id: %s", context, strings.Join(duplicates, ", ")),
			))
		}
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

func (val Validate) validateFunctionReference(function ast.Function) {
	rng := function.ReferenceRange
	if !function.IsString {
		rng = function.Range
	}

	report := func(detail string) {
		val.addDiagnostic(diagnostic.Error(rng, fmt.Sprintf("Function '%s' %s", function.Alias, detail)))
	}

	if !function.IsString {
		report(fmt.Sprintf("must be a string, for example %s", functionReferenceExample))
		return
	}

	path, version, found := strings.Cut(function.Reference, "@")
	if !found {
		report(fmt.Sprintf("is missing an '@version' suffix, which pins the release: %s", functionReferenceExample))
		return
	}

	if !functionReferencePathPattern.MatchString(path) {
		report(fmt.Sprintf("has an invalid path: %q. Expected host.tld/org/name, as in %s", path, functionReferenceExample))
		return
	}

	if !functionReferenceVersionPattern.MatchString(version) {
		report(fmt.Sprintf("has an invalid version: %q. Expected a semver tag led by 'v', as in v0.1.0 or v0.1.0-abc1234", version))
	}
}

func formatFunctionValue(value ast.ParameterValue) string {
	if value.Type == "string" {
		return fmt.Sprintf("%q", fmt.Sprint(value.Value))
	}
	return fmt.Sprint(value.Value)
}
