package validate

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

func (val Validate) ValidateCommands() {
	for _, command := range val.Doc.Commands {
		val.validateSingleCommand(command)
	}
}

func (val Validate) validateSingleCommand(command ast.Command) {
	val.validateSteps(command.Steps, command.Name)

	if used := val.checkIfCommandIsUsed(command); !used {
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

	return false
}

func (val Validate) commandIsUnused(command ast.Command) {
	val.addDiagnostic(utils.CreateWarningDiagnosticFromRange(command.NameRange, "Command is unused"))
}
