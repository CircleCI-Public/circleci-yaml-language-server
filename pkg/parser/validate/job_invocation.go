package validate

import (
	"fmt"
	"sort"
	"strings"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

// InvocationKind distinguishes where a job invocation appears.
type InvocationKind int

const (
	InWorkflow InvocationKind = iota
	InJobGroup
)

// InvocationContext carries the location and identity of where job
// invocations are being validated (either inside a workflow or a job-group)
type InvocationContext struct {
	// Kind indicates whether the invocations are inside a workflow or a job-group.
	Kind InvocationKind

	// JobGroupName is the name of the enclosing job-group definition.
	// Only meaningful when Kind == InJobGroup; empty otherwise.
	JobGroupName string
}

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

// hasBeenRenamed returns true when the invocation has an explicit name: attribute
// (as opposed to the parser's default of StepName == JobName).
func hasBeenRenamed(inv ast.JobInvocation) bool {
	return inv.StepName != inv.JobName
}

// validateDuplicateJobGroupInvocations checks for job-group invocations that
// collide: same group invoked twice with the same name:, or multiple times
// without any name:.
func (val Validate) validateDuplicateJobGroupInvocations(jobInvocations []ast.JobInvocation) {
	type entry struct {
		isRenamed  bool
		name       string
		invocation ast.JobInvocation
	}
	seen := map[string][]entry{}

	for _, inv := range jobInvocations {
		if !val.Doc.DoesJobGroupExist(inv.JobName) {
			continue
		}
		isRenamed := hasBeenRenamed(inv)
		name := ""
		if isRenamed {
			name = inv.StepName
		}
		seen[inv.JobName] = append(seen[inv.JobName], entry{isRenamed: isRenamed, name: name, invocation: inv})
	}

	for groupName, entries := range seen {
		if len(entries) < 2 {
			continue
		}

		namesSeen := map[string]bool{}
		for _, e := range entries {
			if !e.isRenamed {
				val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
					e.invocation.JobNameRange,
					fmt.Sprintf("Job group \"%s\" is invoked multiple times without a \"name\" attribute. Each invocation must have a unique name", groupName),
				))
			} else if namesSeen[e.name] {
				val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
					e.invocation.StepNameRange,
					fmt.Sprintf("Job group \"%s\" is already invoked with the name \"%s\"", groupName, e.name),
				))
			} else {
				namesSeen[e.name] = true
			}
		}
	}
}

// Validates and adds diagnostics for workflow/job-group job invocations.
func (val Validate) validateInvocations(jobInvocations []ast.JobInvocation, ctx InvocationContext) {

	val.validateDuplicateJobGroupInvocations(jobInvocations)
	for _, jobInvocation := range jobInvocations {
		// A job invocation can invoke either a job or job-group, each type requires different validation
		isJobGroup := val.Doc.DoesJobGroupExist(jobInvocation.JobName)
		if isJobGroup {
			val.validateJobGroupInvocation(jobInvocation, ctx)
		} else {
			val.validateSingleJobInvocation(jobInvocation, ctx)
		}

		// Common features between invoking a job and a job-group

		for _, require := range jobInvocation.Requires {
			if !val.doesJobInvocationExist(jobInvocations, require.Name) && !utils.CheckIfMatrixParamIsPartiallyReferenced(require.Name) {
				// Check if the require references a job inside a job-group
				if ownerGroup, found := val.Doc.FindJobGroupContainingJob(require.Name); found {
					if ctx.Kind == InWorkflow {
						val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
							require.Range,
							fmt.Sprintf("\"%s\" is defined inside job group \"%s\", not directly in this workflow", require.Name, ownerGroup)))
						continue
					} else if ctx.Kind == InJobGroup && ownerGroup != ctx.JobGroupName {
						val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
							require.Range,
							fmt.Sprintf("\"%s\" is not a member of this job group", require.Name)))
						continue
					}
				}

				val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
					require.Range,
					fmt.Sprintf("Cannot find declaration for job invocation \"%s\"", require.Name)))
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
	}
}

func (val Validate) validateSingleJobInvocation(jobInvocation ast.JobInvocation, ctx InvocationContext) {
	if ctx.Kind == InJobGroup && jobInvocation.SerialGroup != "" {
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(jobInvocation.SerialGroupRange, "Use of `serial-group` on job invocations inside a job-group is not supported. Please consider using `serial-group` on the job-group instead."))
	}

	// Users can define a job via `type: approval` within a workflow/job-group job invocation
	// https://circleci.com/docs/reference/configuration-reference/#type
	// This is an old artifact that we don't want to expand on anymore.
	if jobInvocation.Type != "" && jobInvocation.Type != "approval" {
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(jobInvocation.TypeRange, fmt.Sprintf("Only jobs with `type: approval` can be defined inline under the `workflows:`/`job-groups:` section. For `type: %s`, define the job in the `jobs:` section instead.", jobInvocation.Type)))
		return
	}

	// This orb check is not needed for job-groups because we don't support job-groups in orbs.
	if val.Doc.IsFromUnfetchableOrb(jobInvocation.JobName) {
		return
	}

	if jobInvocation.Type != "approval" && // Special case: if the job is defined inline via `type: approval`, then it must exist
		!val.Doc.DoesJobExist(jobInvocation.JobName) &&
		!(val.Doc.IsOrbReference(jobInvocation.JobName) && (val.Doc.IsOrbCommand(jobInvocation.JobName, val.Cache) || val.Doc.IsOrbJob(jobInvocation.JobName, val.Cache))) {
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
			jobInvocation.JobInvocationRange,
			fmt.Sprintf("Cannot find declaration for job \"%s\"", jobInvocation.JobName)))
		return
	}

	if !val.Doc.IsOrbReference(jobInvocation.JobName) && !val.Doc.IsBuiltIn(jobInvocation.JobName) {
		val.validateJobInvocationParameters(jobInvocation)
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

// Validates the structure of an invocation of a job-group, which
// does not have all of the same features as a single job invocation.
func (val Validate) validateJobGroupInvocation(jobInvocation ast.JobInvocation, ctx InvocationContext) {
	if ctx.Kind == InJobGroup {
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(jobInvocation.JobNameRange,
			fmt.Sprintf("Job group \"%s\" cannot reference job group \"%s\" -- nesting is not supported", ctx.JobGroupName, jobInvocation.JobName)))
		return // exit early
	}

	// Keys not allowed in job group invocations
	if jobInvocation.HasMatrix {
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(jobInvocation.MatrixRange, "Job group invocations do not support `matrix`"))
	}
	if jobInvocation.OverrideWith != "" {
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(jobInvocation.OverrideWithRange, "Job group invocations do not support use of `override-with`"))
	}
	if jobInvocation.Type != "" {
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(jobInvocation.TypeRange, "Job group invocations do not support use of `type`"))
	}
	if len(jobInvocation.Parameters) > 0 {
		paramNames := make([]string, 0, len(jobInvocation.Parameters))
		for name := range jobInvocation.Parameters {
			paramNames = append(paramNames, fmt.Sprintf("`%s`", name))
		}
		sort.Strings(paramNames) // Since map iteration is not guaranteed to be in order, sort the paramNames
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(jobInvocation.JobInvocationRange,
			fmt.Sprintf("Job group invocations do not support custom parameters, but found: %s", strings.Join(paramNames, ", "))))
	}
	if len(jobInvocation.Context) > 0 {
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(jobInvocation.JobInvocationRange, "Job group invocations do not support use of `context`"))
	}
	if len(jobInvocation.PreSteps) > 0 {
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(jobInvocation.PreStepsRange, "Job group invocations do not support use of `pre-steps`"))
	}
	if len(jobInvocation.PostSteps) > 0 {
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(jobInvocation.PostStepsRange, "Job group invocations do not support use of `post-steps`"))
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
