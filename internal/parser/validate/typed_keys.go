package validate

import (
	"fmt"
	"iter"
	"regexp"
	"slices"
	"strconv"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"

	ast2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/paramref"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/yamltree"
)

// A typedKey is a setting whose value the compiler checks once the
// parameters in it are replaced. Its messages are the compiler's.
type typedKey struct {
	// under is the key the setting belongs to, such as `run`, or "" for a
	// job's own setting.
	under string
	name  string
	// fits is the type a parameter used as the whole value must have:
	// "integer", "boolean", or "string" for any textual type.
	fits    string
	message string
	// check returns why a value the parameter can take isn't allowed, or ""
	// if it is.
	check func(value any) string
}

var checkoutMethodsMessage = `Unsupported checkout method, allowed methods are "blobless", "shallow", and "full"`

var typedKeys = []typedKey{
	{
		name:    "parallelism",
		fits:    "integer",
		message: "parallelism must be an integer or an expression evaluating to an integer",
		check: func(value any) string {
			if n, ok := value.(int); ok && n < 1 {
				return "parallelism must be a positive integer"
			}
			return ""
		},
	},
	{
		name:    "circleci_ip_ranges",
		fits:    "boolean",
		message: "circleci_ip_ranges must be a boolean",
	},
	{
		under:   "run",
		name:    "max_auto_reruns",
		fits:    "integer",
		message: "max_auto_reruns must be an integer or an expression evaluating to an integer",
		check: func(value any) string {
			if n, ok := value.(int); ok && (n < 1 || n > 5) {
				return "max_auto_reruns must be an integer between 1 and 5"
			}
			return ""
		},
	},
	{
		under:   "run",
		name:    "auto_rerun_delay",
		fits:    "string",
		message: "auto_rerun_delay must match " + AUTO_RERUN_DELAY_REGEX.String(),
		check:   checkAutoRerunDelay,
	},
	{
		under:   "setup_remote_docker",
		name:    "docker_layer_caching",
		fits:    "boolean",
		message: "setup_remote_docker: docker_layer_caching must be a boolean",
	},
	{
		under: "machine",
		name:  "docker_layer_caching",
		fits:  "boolean",
		// %s is the executor's name, or <inline> for a job's own machine.
		message: "Executor %s: docker_layer_caching must be a boolean",
	},
	{
		under:   "setup_remote_docker",
		name:    "prefer_same_region",
		fits:    "boolean",
		message: "setup_remote_docker: prefer_same_region must be a boolean",
	},
	{
		under:   "checkout",
		name:    "method",
		fits:    "string",
		message: checkoutMethodsMessage,
		check: func(value any) string {
			if s, ok := value.(string); ok && !slices.Contains(ast2.CheckoutMethods, s) {
				return checkoutMethodsMessage
			}
			return ""
		},
	},
}

var secondsRegex = regexp.MustCompile(`^([0-9]+)s$`)

func checkAutoRerunDelay(value any) string {
	s, ok := value.(string)
	if !ok {
		return ""
	}
	if !AUTO_RERUN_DELAY_REGEX.MatchString(s) {
		return "auto_rerun_delay must match " + AUTO_RERUN_DELAY_REGEX.String()
	}
	if match := secondsRegex.FindStringSubmatch(s); match != nil {
		if seconds, err := strconv.Atoi(match[1]); err == nil && seconds > 600 {
			return "auto_rerun_delay must not exceed 10 minutes (600 seconds)"
		}
	}
	return ""
}

// ValidateTypedKeyReferences checks the settings of jobs, commands and
// executors whose value is only a parameter, such as
// `parallelism: << parameters.n >>`. The parameter must be of a type the setting takes, and its default and every
// argument a job invocation or a step gives it must be a value the setting
// allows. A default or an argument used by several settings is reported
// once.
func (val Validate) ValidateTypedKeyReferences() {
	reported := map[protocol.Range]map[string]bool{}
	for node := range yamltree.Walk(val.Doc.RootNode) {
		if !isPair(node) {
			continue
		}

		keyNode, valueNode := val.Doc.GetKeyValueNodes(node)
		if keyNode == nil || valueNode == nil {
			continue
		}
		// The value is checked first: finding the key's place reads up the
		// tree, which costs more.
		value := val.Doc.GetNodeText(valueNode)
		if !paramref.IsOnlyParameter(value) {
			continue
		}

		key, ok := val.typedKeyOf(node, val.Doc.GetNodeText(keyNode))
		if !ok {
			continue
		}

		for _, diag := range val.checkTypedKeyReference(key, value, val.Doc.NodeToRange(valueNode)) {
			message := diagnostic.MessageText(diag)
			if reported[diag.Range][message] {
				continue
			}
			if reported[diag.Range] == nil {
				reported[diag.Range] = map[string]bool{}
			}
			reported[diag.Range][message] = true
			val.addDiagnostic(diag)
		}
	}
}

func (val Validate) typedKeyOf(pair *sitter.Node, name string) (typedKey, bool) {
	parent := enclosingPair(pair)
	if parent == nil {
		return typedKey{}, false
	}

	for _, key := range typedKeys {
		if key.name != name {
			continue
		}
		if key.under == "machine" {
			if val.pairKey(parent) != "machine" {
				continue
			}
			owner := enclosingPair(parent)
			switch val.pairKey(enclosingPair(owner)) {
			case "jobs":
				key.message = fmt.Sprintf(key.message, "<inline>")
			case "executors":
				key.message = fmt.Sprintf(key.message, val.pairKey(owner))
			default:
				continue
			}
			return key, true
		}
		if key.under != "" && key.under == val.pairKey(parent) ||
			key.under == "" && val.pairKey(enclosingPair(parent)) == "jobs" {
			return key, true
		}
	}

	return typedKey{}, false
}

func (val Validate) checkTypedKeyReference(key typedKey, reference string, rng protocol.Range) []protocol.Diagnostic {
	fullName, name := paramref.ExtractName(reference)

	// Each inline orb is checked on its own, so only this document's jobs,
	// commands and executors are looked at.
	var params map[string]ast2.Parameter
	var arguments []ast2.ParameterValue
	if job, ok := val.jobAt(rng.Start); ok {
		params = job.Parameters
		arguments = val.jobArguments(job.Name, name)
	} else if command, ok := val.commandAt(rng.Start); ok {
		params = command.Parameters
		arguments = val.commandArguments(command.Name, name)
	} else if executor, ok := val.executorAt(rng.Start); ok {
		params = executor.GetParameters()
	} else {
		return nil
	}

	if strings.HasPrefix(fullName, "pipeline.") {
		params = val.Doc.PipelineParameters
		arguments = nil
	}
	param := params[name]

	// An undefined parameter is reported by CheckIfParamsExist.
	if param == nil {
		return nil
	}

	if !parameterFits(key.fits, param.GetType()) {
		return []protocol.Diagnostic{diagnostic.Error(rng, fmt.Sprintf(
			"%s, but parameter %s is of type %s", key.message, name, param.GetType()))}
	}

	if key.check == nil {
		return nil
	}

	var diags []protocol.Diagnostic

	if value, ok := defaultOf(param); ok {
		if message := key.check(value); message != "" {
			diags = append(diags, diagnostic.Error(param.GetDefaultRange(), fmt.Sprintf(
				"%s: parameter %s is used for %s, and defaults to %v", message, name, key.name, value)))
		}
	}

	for _, argument := range arguments {
		if s, ok := argument.Value.(string); ok && paramref.ContainsReference(s) {
			continue
		}
		if message := key.check(argument.Value); message != "" {
			diags = append(diags, diagnostic.Error(argument.ValueRange, fmt.Sprintf(
				"%s: parameter %s is used for %s, and is given %v", message, name, key.name, argument.Value)))
		}
	}
	return diags
}

func parameterFits(fits, paramType string) bool {
	if fits == "string" {
		return slices.Contains([]string{"string", "enum", "env_var_name"}, paramType)
	}
	return fits == paramType
}

func defaultOf(param ast2.Parameter) (any, bool) {
	if !param.IsOptional() {
		return nil, false
	}

	switch param := param.(type) {
	case ast2.IntegerParameter:
		return param.Default, true
	case ast2.StringParameter:
		return param.Default, !paramref.ContainsReference(param.Default)
	case ast2.EnumParameter:
		return param.Default, true
	case ast2.EnvVariableParameter:
		return param.Default, true
	}
	return nil, false
}

func (val Validate) jobAt(pos protocol.Position) (ast2.Job, bool) {
	for _, job := range val.Doc.Jobs {
		if position.InRange(job.Range, pos) {
			return job, true
		}
	}
	return ast2.Job{}, false
}

func (val Validate) executorAt(pos protocol.Position) (ast2.Executor, bool) {
	for _, executor := range val.Doc.Executors {
		if position.InRange(executor.GetRange(), pos) {
			return executor, true
		}
	}
	return nil, false
}

func (val Validate) commandAt(pos protocol.Position) (ast2.Command, bool) {
	for _, command := range val.Doc.Commands {
		if position.InRange(command.Range, pos) {
			return command, true
		}
	}
	return ast2.Command{}, false
}

// A caller is a document that can call this document's jobs and commands,
// and the names it calls one by.
type caller struct {
	doc   parser.YamlDocument
	names []string
}

// callers returns the documents that call the job or command name: this one,
// and for an inline orb the config, which calls it by `orb/name` or through
// an alias of that.
func (val Validate) callers(name string, aliases func(parser.YamlDocument) map[string]ast2.Alias) []caller {
	callers := []caller{{doc: val.Doc, names: []string{name}}}
	if val.Outer == nil {
		return callers
	}

	outer := caller{doc: *val.Outer, names: []string{val.OrbName + "/" + name}}
	for _, alias := range aliases(*val.Outer) {
		if alias.Target == outer.names[0] {
			outer.names = append(outer.names, alias.Name)
		}
	}
	return append(callers, outer)
}

func jobAliases(doc parser.YamlDocument) map[string]ast2.Alias     { return doc.Aliases.Jobs }
func commandAliases(doc parser.YamlDocument) map[string]ast2.Alias { return doc.Aliases.Commands }

// invocations yields a document's workflow jobs and job group members, in
// place rather than copied.
func invocations(doc *parser.YamlDocument) iter.Seq[*ast2.JobInvocation] {
	return func(yield func(*ast2.JobInvocation) bool) {
		for _, workflow := range doc.Workflows {
			for i := range workflow.JobInvocations {
				if !yield(&workflow.JobInvocations[i]) {
					return
				}
			}
		}
		for _, group := range doc.JobGroups {
			for i := range group.JobInvocations {
				if !yield(&group.JobInvocations[i]) {
					return
				}
			}
		}
	}
}

// jobArguments returns the values the workflows and job groups give a job's
// parameter, directly or through a matrix.
func (val Validate) jobArguments(jobName, paramName string) []ast2.ParameterValue {
	var arguments []ast2.ParameterValue
	for _, caller := range val.callers(jobName, jobAliases) {
		for invocation := range invocations(&caller.doc) {
			if !slices.Contains(caller.names, invocation.JobName) {
				continue
			}
			if argument, ok := invocation.Parameters[paramName]; ok {
				arguments = append(arguments, argument)
			}
			for _, matrix := range invocation.MatrixParams[paramName] {
				if values, ok := matrix.Value.([]ast2.ParameterValue); ok {
					arguments = append(arguments, values...)
				}
			}
		}
	}
	return arguments
}

// commandArguments returns the values the steps calling a command give its
// parameter.
func (val Validate) commandArguments(commandName, paramName string) []ast2.ParameterValue {
	var arguments []ast2.ParameterValue
	for _, caller := range val.callers(commandName, commandAliases) {
		collect := func(step ast2.Step) bool {
			if named, ok := step.(ast2.NamedStep); ok && slices.Contains(caller.names, named.Name) {
				if argument, ok := named.Parameters[paramName]; ok {
					arguments = append(arguments, argument)
				}
			}
			return false
		}

		for _, job := range caller.doc.Jobs {
			anyStep(job.Steps, collect)
		}
		for _, command := range caller.doc.Commands {
			anyStep(command.Steps, collect)
		}
		for invocation := range invocations(&caller.doc) {
			anyStep(invocation.PreSteps, collect)
			anyStep(invocation.PostSteps, collect)
		}
	}
	return arguments
}
