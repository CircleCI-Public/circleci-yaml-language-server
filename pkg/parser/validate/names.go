package validate

import (
	"fmt"

	"github.com/circleci/circleci-yaml-language-server/pkg/utils"
)

func (val Validate) CheckNames() {
	for _, workflow := range val.Doc.Workflows {
		for _, job := range val.Doc.Jobs {
			if workflow.Name == job.Name {
				val.addDiagnostic(utils.CreateWarningDiagnosticFromRange(
					workflow.NameRange,
					fmt.Sprintf("The name \"%s\" is already used to define a job. You might want to use a different name them to avoid confusion.", workflow.Name)))
				val.addDiagnostic(utils.CreateWarningDiagnosticFromRange(
					job.NameRange,
					fmt.Sprintf("The name \"%s\" is already used to define a workflow. You might want to use a different name them to avoid confusion.", job.Name)))
			}
		}

		for _, command := range val.Doc.Commands {
			if workflow.Name == command.Name {
				val.addDiagnostic(utils.CreateWarningDiagnosticFromRange(
					workflow.NameRange,
					fmt.Sprintf("The name \"%s\" is already used to define a command. You might want to use a different name them to avoid confusion.", workflow.Name)))
				val.addDiagnostic(utils.CreateWarningDiagnosticFromRange(
					command.NameRange,
					fmt.Sprintf("The name \"%s\" is already used to define a workflow. You might want to use a different name them to avoid confusion.", command.Name)))
			}
		}
	}

	for _, job := range val.Doc.Jobs {
		for _, command := range val.Doc.Commands {
			if job.Name == command.Name {
				val.addDiagnostic(utils.CreateWarningDiagnosticFromRange(
					job.NameRange,
					fmt.Sprintf("The name \"%s\" is already used to define a command. You might want to use a different name them to avoid confusion.", job.Name)))
				val.addDiagnostic(utils.CreateWarningDiagnosticFromRange(
					command.NameRange,
					fmt.Sprintf("The name \"%s\" is already used to define a job. You might want to use a different name them to avoid confusion.", command.Name)))
			}
		}
	}
}
