package validate

import (
	"fmt"
	"slices"

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
