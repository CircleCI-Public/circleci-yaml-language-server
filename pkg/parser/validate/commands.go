package validate

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

func (val Validate) ValidateCommands() {
	if len(val.Doc.Commands) == 0 && !utils.IsDefaultRange(val.Doc.CommandsRange) {
		val.addDiagnostic(
			utils.CreateEmptyAssignationWarning(val.Doc.CommandsRange),
		)

		return
	}

	for _, command := range val.Doc.Commands {
		val.validateSingleCommand(command)
	}
}

func (val Validate) validateSingleCommand(command ast.Command) {
	val.validateSteps(command.Steps, command.Name, command.Parameters)

	// Local orbs do not need unused checks because those checks collides with the overall YAML unused checks
	if val.IsLocalOrb && !val.checkIfCommandIsUsed(command) {
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
		for _, jobRef := range workflow.JobRefs {
			steps := jobRef.PostSteps
			steps = append(steps, jobRef.PreSteps...)

			if val.checkIfStepsContainStep(steps, command.Name) {
				return true
			}
		}
	}

	return false
}

func (val Validate) commandIsUnused(command ast.Command) {
	val.addDiagnostic(utils.CreateWarningDiagnosticFromRange(command.NameRange, "Command is unused"))
}
