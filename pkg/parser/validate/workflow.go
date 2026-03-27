package validate

import (
	"fmt"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

var TerminalJobStatuses = []string{"success", "failed", "canceled", "not_run", "unauthorized"}

func (val Validate) ValidateWorkflows() {
	for _, workflow := range val.Doc.Workflows {
		val.validateSingleWorkflow(workflow)
	}
}

func (val Validate) validateSingleWorkflow(workflow ast.Workflow) error {
	if workflow.HasMaxAutoReruns {
		if workflow.MaxAutoReruns < 1 {
			val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(workflow.MaxAutoRerunsRange, "Must be greater than or equal to 1"))
		} else if workflow.MaxAutoReruns > 5 {
			val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(workflow.MaxAutoRerunsRange, "Must be less than or equal to 5"))
		}
	}

	for _, jobInvocation := range workflow.JobInvocations {
		if val.Doc.IsFromUnfetchableOrb(jobInvocation.JobName) {
			continue
		}

		isApprovalJob := jobInvocation.Type == "approval"
		if isApprovalJob {
			continue
		}

		jobTypeIsDefined := jobInvocation.Type != ""
		if jobTypeIsDefined {
			val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(jobInvocation.TypeRange, fmt.Sprintf("Only jobs with `type: approval` can be defined inline under the `workflows:` section. For `type: %s`, define the job in the `jobs:` section instead.", jobInvocation.Type)))
			continue
		}

		if !val.Doc.DoesJobExist(jobInvocation.JobName) &&
			!(val.Doc.IsOrbReference(jobInvocation.JobName) && (val.Doc.IsOrbCommand(jobInvocation.JobName, val.Cache) || val.Doc.IsOrbJob(jobInvocation.JobName, val.Cache))) {
			val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
				jobInvocation.JobInvocationRange,
				fmt.Sprintf("Cannot find declaration for job %s", jobInvocation.JobName)))
		}

		if !val.Doc.IsOrbReference(jobInvocation.JobName) && !val.Doc.IsBuiltIn(jobInvocation.JobName) {
			val.validateWorkflowParameters(jobInvocation, jobInvocation.JobName, jobInvocation.JobInvocationRange)
		}
		for _, require := range jobInvocation.Requires {
			if !val.doesJobInvocationExist(workflow, require.Name) && !utils.CheckIfMatrixParamIsPartiallyReferenced(require.Name) {
				val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
					require.Range,
					fmt.Sprintf("Cannot find declaration for job invocation %s", require.Name)))
			}

			if requireHasAllTerminalStatuses(require.Status) {
				// Use " terminal" for multi-line arrays so there's a space after the colon.
				// "terminal" for inline arrays since we're replacing an array that is
				// already spaced after the colon. e.g.
				//
				// Inline:
				// Before: - job_name: [inline-array]
				// After:  - job_name: terminal
				//
				// Vs multi-line:
				// Before:
				// - job_name:
				//   - success
				//
				// After:
				// - job_name: terminal
				newText := "terminal"
				if require.StatusRange.Start.Line != require.StatusRange.End.Line {
					newText = " terminal"
				}
				codeAction := utils.CreateCodeActionTextEdit(
					"Simplify these statuses to 'terminal'",
					val.Doc.URI,
					[]protocol.TextEdit{
						{
							NewText: newText,
							Range:   require.StatusRange,
						},
					},
					true, // preferred
				)
				val.addDiagnostic(
					protocol.Diagnostic{
						Range:    require.StatusRange,
						Message:  fmt.Sprintf("Statuses: '%v' can be simplified to just 'terminal'", require.Status),
						Severity: protocol.DiagnosticSeverityHint,
						Data:     []protocol.CodeAction{codeAction},
					},
				)
			}
		}

		if cachedFile := val.Cache.FileCache.GetFile(val.Doc.URI); val.Context.Api.Token != "" &&
			cachedFile != nil && cachedFile.Project.OrganizationName != "" {
			for _, context := range jobInvocation.Context {
				if context.Text != "org-global" && val.Cache.ContextCache.GetOrganizationContext(cachedFile.Project.OrganizationId, context.Text) == nil {
					val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
						context.Range,
						fmt.Sprintf("Context %s does not exist", context.Text)))
				}
			}
		}
	}

	val.validateDAG(workflow)

	return nil
}

func (val Validate) doesJobInvocationExist(workflow ast.Workflow, requireName string) bool {
	for _, jobInvocation := range workflow.JobInvocations {
		if jobInvocation.JobName == requireName || jobInvocation.StepName == requireName {
			return true
		}
	}
	return false
}

func (val Validate) validateWorkflowParameters(jobInvocation ast.JobInvocation, stepName string, stepRange protocol.Range) {
	definedParams := val.Doc.GetDefinedParams(stepName, val.Cache)

	for _, definedParam := range definedParams {
		_, okMatrix := jobInvocation.MatrixParams[definedParam.GetName()]
		_, okParams := jobInvocation.Parameters[definedParam.GetName()]

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
			for _, param := range jobInvocation.MatrixParams[definedParam.GetName()] {
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
			val.checkParamSimpleType(jobInvocation.Parameters[definedParam.GetName()], stepName, definedParam)
		}
	}

	for _, param := range jobInvocation.Parameters {
		if definedParams[param.Name] == nil {
			val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
				param.Range,
				fmt.Sprintf("Parameter %s is not defined in %s", param.Name, stepName)),
			)
		}
	}
}

func (val Validate) validateDAG(workflow ast.Workflow) {
	nodes_in_cycle := isValidDAG(workflow.JobsDAG)

	for _, node := range nodes_in_cycle {
		for _, jobInvocation := range workflow.JobInvocations {
			if jobInvocation.JobName == node {
				val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
					jobInvocation.JobNameRange,
					fmt.Sprintf("The job `%s` is part of a cycle", node)))
			}
		}
	}
}

func requireHasAllTerminalStatuses(statuses []string) bool {
	if len(statuses) != len(TerminalJobStatuses) {
		return false
	}

	terminalSet := make(map[string]bool)
	for _, status := range TerminalJobStatuses {
		terminalSet[status] = false
	}

	for _, s := range statuses {
		if _, ok := terminalSet[s]; ok {
			terminalSet[s] = true
		} else {
			return false
		}
	}

	for _, found := range terminalSet {
		if !found {
			return false
		}
	}

	return true
}
