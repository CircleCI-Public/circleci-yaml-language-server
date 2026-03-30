package validate

import (
	"fmt"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

func (val Validate) doesJobInvocationExist(jobInvocations []ast.JobInvocation, requireName string) bool {
	for _, jobInvocation := range jobInvocations {
		if jobInvocation.JobName == requireName || jobInvocation.StepName == requireName {
			return true
		}
	}
	return false
}

func (val Validate) validateJobInvocationParameters(jobInvocation ast.JobInvocation) {
	jobName := jobInvocation.JobName
	jobRange := jobInvocation.JobInvocationRange
	definedParams := val.Doc.GetDefinedParams(jobName, val.Cache)

	for _, definedParam := range definedParams {
		_, okMatrix := jobInvocation.MatrixParams[definedParam.GetName()]
		_, okParams := jobInvocation.Parameters[definedParam.GetName()]

		if !okMatrix && !okParams && !definedParam.IsOptional() {
			val.addDiagnostic(
				utils.CreateErrorDiagnosticFromRange(
					jobRange,
					fmt.Sprintf("Parameter %s is required for %s", definedParam.GetName(), jobName),
				),
			)
			continue
		}

		if okMatrix {
			for _, param := range jobInvocation.MatrixParams[definedParam.GetName()] {
				if param.Type == "enum" {
					for _, value := range param.Value.([]ast.ParameterValue) {
						val.checkParamSimpleType(value, jobName, definedParam)
					}
				} else if param.Type != "alias" {
					val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
						param.Range,
						fmt.Sprintf("Parameter %s is not an enum of values", param.Name)),
					)
				}
			}
		} else if okParams {
			val.checkParamSimpleType(jobInvocation.Parameters[definedParam.GetName()], jobName, definedParam)
		}
	}

	for _, param := range jobInvocation.Parameters {
		if definedParams[param.Name] == nil {
			val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
				param.Range,
				fmt.Sprintf("Parameter %s is not defined in %s", param.Name, jobName)),
			)
		}
	}
}

// Validates and adds diagnostics for a list of job invocations.
func (val Validate) validateInvocations(jobInvocations []ast.JobInvocation) {
	for _, jobInvocation := range jobInvocations {
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
			val.validateJobInvocationParameters(jobInvocation)
		}

		for _, require := range jobInvocation.Requires {
			if !val.doesJobInvocationExist(jobInvocations, require.Name) && !utils.CheckIfMatrixParamIsPartiallyReferenced(require.Name) {
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
}

func (val Validate) validateDAG(invocations []ast.JobInvocation, dag map[string][]string) {
	nodesInCycle := isValidDAG(dag)

	for _, node := range nodesInCycle {
		for _, invocation := range invocations {
			if invocation.JobName == node {
				val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
					invocation.JobNameRange,
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
