package validate

import (
	"fmt"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func (val Validate) ValidateWorkflows() {
	for _, workflow := range val.Doc.Workflows {
		val.validateSingleWorkflow(workflow)
	}
}

func (val Validate) validateSingleWorkflow(workflow ast.Workflow) error {
	for _, jobRef := range workflow.JobRefs {
		isApprovalJob := jobRef.Type == "approval"
		if isApprovalJob {
			continue
		}

		jobTypeIsDefined := jobRef.Type != ""
		if jobTypeIsDefined {
			val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(jobRef.TypeRange, "Type can only be \"approval\""))
			continue
		}

		if !val.Doc.DoesJobExist(jobRef.JobName) && !val.Doc.IsOrbReference(jobRef.JobName) {
			val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
				jobRef.JobRefRange,
				fmt.Sprintf("Cannot find declaration for job %s", jobRef.JobName)))
		}

		if !val.Doc.IsOrbReference(jobRef.JobName) && !val.Doc.IsBuiltIn(jobRef.JobName) {
			val.validateWorkflowParameters(jobRef, jobRef.JobName, jobRef.JobRefRange)
		}
	}

	val.validateDAG(workflow)

	return nil
}

func (val Validate) validateWorkflowParameters(jobRef ast.JobRef, stepName string, stepRange protocol.Range) {
	definedParams := val.Doc.GetDefinedParams(stepName)

	for _, definedParam := range definedParams {
		_, okMatrix := jobRef.MatrixParams[definedParam.GetName()]
		_, okParams := jobRef.Parameters[definedParam.GetName()]

		if !okMatrix && !okParams && !definedParam.IsOptional() {
			val.addDiagnostic(
				utils.CreateErrorDiagnosticFromRange(
					stepRange,
					fmt.Sprintf("Parameter %s is required for %s", definedParam.GetName(), stepName),
				),
			)
			continue
		}

		if okMatrix {
			for _, param := range jobRef.MatrixParams[definedParam.GetName()] {
				if param.Type == "enum" {
					for _, value := range param.Value.([]ast.ParameterValue) {
						val.checkParamSimpleType(value, stepName, definedParam)
					}
				} else if param.Type != "alias" {
					val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
						param.Range,
						fmt.Sprintf("Parameter %s is not an enum of values", param.Name)),
					)
				}
			}
		} else if okParams {
			val.checkParamSimpleType(jobRef.Parameters[definedParam.GetName()], stepName, definedParam)
		}
	}
}

func (val Validate) validateDAG(workflow ast.Workflow) {
	nodes_in_cycle := isValidDAG(workflow.JobsDAG)

	for _, node := range nodes_in_cycle {
		for _, jobRef := range workflow.JobRefs {
			if jobRef.JobName == node {
				val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
					jobRef.JobNameRange,
					fmt.Sprintf("The job `%s` is part of a cycle", node)))
			}
		}
	}
}
