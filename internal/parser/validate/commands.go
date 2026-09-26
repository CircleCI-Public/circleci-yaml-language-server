package validate

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func (val Validate) ValidateCommands() {
	if len(val.Doc.Commands) == 0 && !position.IsDefaultRange(val.Doc.CommandsRange) {
		val.addDiagnostic(
			diagnostic.EmptyAssignationWarning(val.Doc.CommandsRange),
		)

		return
	}

	for _, command := range val.Doc.Commands {
		val.validateSingleCommand(command)
	}
	val.validateCommandRecursion()
}

// validateCommandRecursion reports each command that calls itself, directly
// or through other commands, which the compiler would expand forever.
func (val Validate) validateCommandRecursion() {
	names := slices.Sorted(maps.Keys(val.Doc.Commands))
	calls := map[string][]string{}
	for _, caller := range names {
		for _, callee := range names {
			// A built-in step's `name:` is not a call.
			isCall := func(step ast.Step) bool {
				named, ok := step.(ast.NamedStep)
				return ok && named.Name == callee
			}
			if anyStep(val.Doc.Commands[caller].Steps, isCall) {
				calls[caller] = append(calls[caller], callee)
			}
		}
	}

	for _, name := range names {
		path, ok := callPathBack(calls, name)
		if !ok {
			continue
		}
		message := fmt.Sprintf("Infinite loop detected: command %s calls itself", name)
		if len(path) > 0 {
			message += " through " + strings.Join(path, ", then ")
		}
		val.addDiagnostic(diagnostic.Error(val.Doc.Commands[name].NameRange, message))
	}
}

// callPathBack finds the shortest chain of calls from name back to itself,
// and returns the commands in between.
func callPathBack(calls map[string][]string, name string) ([]string, bool) {
	parent := map[string]string{}
	queue := []string{name}
	for len(queue) > 0 {
		caller := queue[0]
		queue = queue[1:]
		for _, callee := range calls[caller] {
			if callee == name {
				path := []string{}
				for step := caller; step != name; step = parent[step] {
					path = append([]string{step}, path...)
				}
				return path, true
			}
			if _, seen := parent[callee]; !seen {
				parent[callee] = caller
				queue = append(queue, callee)
			}
		}
	}
	return nil, false
}

// primitiveSteps are the built-in steps a command can be named after, taking
// their place.
var primitiveSteps = []string{
	"run",
	"checkout",
	"setup_remote_docker",
	"save_cache",
	"restore_cache",
	"deploy",
	"store_artifacts",
	"store_test_results",
	"persist_to_workspace",
	"attach_workspace",
	"add_ssh_keys",
	"install_signing_bundle",
}

func (val Validate) validateSingleCommand(command ast.Command) {
	val.validateSteps(command.Steps, command.Name, command.Parameters)

	if slices.Contains(primitiveSteps, command.Name) {
		path := "commands"
		if val.Doc.LocalOrbName != "" {
			path = "orbs." + val.Doc.LocalOrbName + ".commands"
		}
		val.addDiagnostic(diagnostic.Warning(command.NameRange, fmt.Sprintf(
			"Command '%s' in %s.%s shadows built-in CircleCI command '%s'",
			command.Name, path, command.Name, command.Name)))
	}

	// Local orbs do not need unused checks because those checks collides with the overall YAML unused checks
	if !val.IsLocalOrb && !val.checkIfCommandIsUsed(command) {
		val.commandIsUnused(command)
	}
}

func (val Validate) checkIfCommandIsUsed(command ast.Command) bool {
	for _, definedCommand := range val.Doc.Commands {
		if val.checkIfStepsContainStep(definedCommand.Steps, command.Name) {
			return true
		}
	}

	for _, job := range val.Doc.Jobs {
		if val.checkIfStepsContainStep(job.Steps, command.Name) {
			return true
		}
	}

	for _, workflow := range val.Doc.Workflows {
		for _, jobInvocation := range workflow.JobInvocations {
			steps := jobInvocation.PostSteps
			steps = append(steps, jobInvocation.PreSteps...)

			if val.checkIfStepsContainStep(steps, command.Name) {
				return true
			}

			if anyParamStep(jobInvocation.Parameters, func(step ast.Step) bool { return step.GetName() == command.Name }) {
				return true
			}
		}
	}

	return false
}

func (val Validate) commandIsUnused(command ast.Command) {
	val.addDiagnostic(diagnostic.Warning(command.NameRange, "Command is unused"))
}
