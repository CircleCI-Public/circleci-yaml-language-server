package validate

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	"go.lsp.dev/protocol"

	ast2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/codeaction"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/paramref"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
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

func (val Validate) doesJobInvocationExist(jobInvocations []ast2.JobInvocation, requireName string) bool {
	for _, jobInvocation := range jobInvocations {
		names := append([]string{jobInvocation.JobName, jobInvocation.StepName, jobInvocation.MatrixAlias}, jobInvocation.MatrixNames...)
		if slices.ContainsFunc(names, func(name string) bool { return invocationNameMatches(name, requireName) }) {
			return true
		}
	}
	return false
}

// invocationNameMatches reports whether a job's name is the name a `requires`
// gives. A name holding a reference, such as `deploy-<< pipeline.git.branch >>`,
// is only known once the pipeline runs.
func invocationNameMatches(name, requireName string) bool {
	return name == requireName || paramref.CouldExpandTo(name, requireName)
}

func (val Validate) validateJobInvocationParameters(jobInvocation ast2.JobInvocation) {
	jobName := jobInvocation.JobName
	jobRange := jobInvocation.JobInvocationRange
	definedParams := val.Doc.GetDefinedParams(jobName, parser.JobEntity, val.Cache)

	for _, definedParam := range definedParams {
		_, okMatrix := jobInvocation.MatrixParams[definedParam.GetName()]
		_, okParams := jobInvocation.Parameters[definedParam.GetName()]

		if !okMatrix && !okParams && !definedParam.IsOptional() {
			val.addDiagnostic(
				diagnostic.Error(
					jobRange,
					fmt.Sprintf("Parameter %s is required for %s", definedParam.GetName(), jobName),
				),
			)
			continue
		}

		if okMatrix {
			for _, param := range jobInvocation.MatrixParams[definedParam.GetName()] {
				if param.Type == "enum" {
					for _, value := range param.Value.([]ast2.ParameterValue) {
						val.checkParamSimpleType(value, jobName, definedParam)
					}
				} else if param.Type != "alias" && param.Type != "null" {
					val.addDiagnostic(diagnostic.Error(
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
			val.addDiagnostic(diagnostic.Error(
				param.Range,
				fmt.Sprintf("Parameter %s is not defined in %s", param.Name, jobName)),
			)
		}
	}

	for name, values := range jobInvocation.MatrixParams {
		if definedParams[name] == nil && len(values) > 0 {
			val.addDiagnostic(diagnostic.Error(
				values[0].Range,
				fmt.Sprintf("Parameter %s is not defined in %s", name, jobName)),
			)
		}
	}
}

// hasBeenRenamed returns true when the invocation has an explicit name: attribute
// (as opposed to the parser's default of StepName == JobName).
func hasBeenRenamed(inv ast2.JobInvocation) bool {
	return inv.StepName != inv.JobName
}

// validateDuplicateJobGroupInvocations checks for job-group invocations that
// collide: same group invoked twice with the same name:, or more than once
// without any name:. One invocation without a name takes the group's name.
func (val Validate) validateDuplicateJobGroupInvocations(jobInvocations []ast2.JobInvocation) {
	type entry struct {
		isRenamed  bool
		name       string
		invocation ast2.JobInvocation
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

		unnamed := 0
		for _, e := range entries {
			if !e.isRenamed {
				unnamed++
			}
		}
		namesSeen := map[string]bool{}
		for _, e := range entries {
			switch {
			case !e.isRenamed:
				if unnamed > 1 {
					val.addDiagnostic(diagnostic.Error(
						e.invocation.JobNameRange,
						fmt.Sprintf("Job group \"%s\" is invoked multiple times without a \"name\" attribute. Each invocation must have a unique name", groupName),
					))
				}
			case namesSeen[e.name]:
				val.addDiagnostic(diagnostic.Error(
					e.invocation.StepNameRange,
					fmt.Sprintf("Job group \"%s\" is already invoked with the name \"%s\"", groupName, e.name),
				))
			default:
				namesSeen[e.name] = true
			}
		}
	}
}

// Validates and adds diagnostics for workflow/job-group job invocations.
func (val Validate) validateInvocations(jobInvocations []ast2.JobInvocation, ctx InvocationContext) {

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

		// Every job takes pre-steps and post-steps, as steps parameters.
		val.validateSteps(jobInvocation.PreSteps, "", map[string]ast2.Parameter{})
		val.validateSteps(jobInvocation.PostSteps, "", map[string]ast2.Parameter{})
		val.warnNullBodySteps(jobInvocation.PreSteps)
		val.warnNullBodySteps(jobInvocation.PostSteps)

		for _, require := range jobInvocation.Requires {
			if !val.doesJobInvocationExist(jobInvocations, require.Name) && !paramref.IsMatrixPartiallyReferenced(require.Name) {
				// Check if the require references a job inside a job-group
				if ownerGroup, found := val.Doc.FindJobGroupContainingJob(require.Name); found {
					if ctx.Kind == InWorkflow {
						val.addDiagnostic(diagnostic.Error(
							require.Range,
							fmt.Sprintf("\"%s\" is defined inside job group \"%s\", not directly in this workflow", require.Name, ownerGroup)))
						continue
					} else if ctx.Kind == InJobGroup && ownerGroup != ctx.JobGroupName {
						val.addDiagnostic(diagnostic.Error(
							require.Range,
							fmt.Sprintf("\"%s\" is not a member of this job group", require.Name)))
						continue
					}
				}

				val.addDiagnostic(diagnostic.Error(
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
				codeAction := codeaction.TextEdit(
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
						Message:  protocol.String(fmt.Sprintf("Statuses: '%v' can be simplified to just 'terminal'", require.Status)),
						Severity: protocol.DiagnosticSeverityHint,
						Data:     codeaction.Data([]protocol.CodeAction{codeAction}),
					},
				)
			}
		}
	}
}

func (val Validate) validateSingleJobInvocation(jobInvocation ast2.JobInvocation, ctx InvocationContext) {
	if ctx.Kind == InJobGroup && jobInvocation.SerialGroup != "" {
		val.addDiagnostic(diagnostic.Error(jobInvocation.SerialGroupRange, "Use of `serial-group` on job invocations inside a job-group is not supported. Please consider using `serial-group` on the job-group instead."))
	}

	// Users can define a job via `type: approval` within a workflow/job-group job invocation
	// https://circleci.com/docs/reference/configuration-reference/#type
	// This is an old artifact that we don't want to expand on anymore.
	if jobInvocation.Type != "" && jobInvocation.Type != "approval" {
		val.addDiagnostic(diagnostic.Error(jobInvocation.TypeRange, fmt.Sprintf("Only jobs with `type: approval` can be defined inline under the `workflows:`/`job-groups:` section. For `type: %s`, define the job in the `jobs:` section instead.", jobInvocation.Type)))
		return
	}

	if jobInvocation.MatrixJobCount > parser.MaxMatrixJobs {
		val.addDiagnostic(diagnostic.Error(matrixKeyRange(jobInvocation), fmt.Sprintf(
			"The %s build matrix expands to %d jobs. Matrices cannot generate more than %d jobs.",
			jobInvocation.MatrixAlias, jobInvocation.MatrixJobCount, parser.MaxMatrixJobs)))
	}

	for name, values := range jobInvocation.MatrixParams {
		for _, value := range values {
			if value.Type == "null" {
				val.addDiagnostic(diagnostic.Warning(value.Range,
					fmt.Sprintf("Matrix parameter '%s' is null; the matrix will produce 0 jobs", name)))
			}
		}
	}

	if jobInvocation.MatrixIsSingleCombination {
		val.addDiagnostic(diagnostic.Warning(matrixKeyRange(jobInvocation),
			"This matrix is declared with a single value for every parameter, so it always "+
				"produces exactly one job. Consider not using a matrix here."))
	}

	if jobInvocation.Type == "approval" {
		val.validateApprovalInvocation(jobInvocation)
		val.validateInvocationContexts(jobInvocation)
		return
	}

	// This orb check is not needed for job-groups because we don't support job-groups in orbs.
	if val.Doc.IsFromUnfetchableOrb(jobInvocation.JobName, val.Cache) {
		return
	}

	if !val.Doc.DoesJobExist(jobInvocation.JobName) && !val.Doc.IsOrbJob(jobInvocation.JobName, val.Cache) {
		message := fmt.Sprintf("Cannot find declaration for job \"%s\"", jobInvocation.JobName)
		if val.Doc.DoesCommandExist(jobInvocation.JobName) || val.Doc.IsOrbCommand(jobInvocation.JobName, val.Cache) {
			message = fmt.Sprintf("%s is a command, not a job: a workflow runs jobs", jobInvocation.JobName)
		}
		val.addDiagnostic(diagnostic.Error(jobInvocation.JobInvocationRange, message))
		return
	}

	if !val.Doc.IsBuiltIn(jobInvocation.JobName) {
		val.validateJobInvocationParameters(jobInvocation)
	}

	val.validateInvocationContexts(jobInvocation)
}

// matrixKeyRange is the range of an invocation's `matrix` key.
func matrixKeyRange(jobInvocation ast2.JobInvocation) protocol.Range {
	rng := protocol.Range{Start: jobInvocation.MatrixRange.Start, End: jobInvocation.MatrixRange.Start}
	rng.End.Character += uint32(len("matrix"))
	return rng
}

// validateInvocationContexts checks that each context exists, when the
// organization's contexts are known.
func (val Validate) validateInvocationContexts(jobInvocation ast2.JobInvocation) {
	if cachedFile := val.Cache.FileCache.GetFile(val.Doc.URI); val.Context.Api.Token != "" &&
		cachedFile != nil && cachedFile.Project.OrganizationName != "" &&
		val.Cache.ContextCache.IsOrganizationContextListLoaded(cachedFile.Project.OrganizationId) {
		for _, context := range jobInvocation.Context {
			if context.Text != "org-global" && val.Cache.ContextCache.ResolveWorkflowContext(
				cachedFile.Project.OrganizationId,
				cachedFile.Project.OrganizationSlug,
				context.Text,
			) == nil {
				val.addDiagnostic(diagnostic.Error(
					context.Range,
					fmt.Sprintf("Context %s does not exist", context.Text)))
			}
		}
	}
}

// validateApprovalInvocation checks an invocation with `type: approval`, which
// defines an approval job there and then. An approval job has no parameters,
// so any other key is ignored.
func (val Validate) validateApprovalInvocation(jobInvocation ast2.JobInvocation) {
	if val.Doc.DoesJobExist(jobInvocation.JobName) {
		val.addDiagnostic(diagnostic.Warning(jobInvocation.JobNameRange, fmt.Sprintf(
			"'%s' is invoked here with type: approval, which shadows the job definition of the same name; "+
				"its own steps will not run for this invocation", jobInvocation.JobName)))
	}

	ignored := map[string]protocol.Range{}
	for name, parameter := range jobInvocation.Parameters {
		ignored[name] = parameter.Range
	}
	if !position.IsDefaultRange(jobInvocation.PreStepsRange) {
		ignored["pre-steps"] = jobInvocation.PreStepsRange
	}
	if !position.IsDefaultRange(jobInvocation.PostStepsRange) {
		ignored["post-steps"] = jobInvocation.PostStepsRange
	}
	for _, name := range slices.Sorted(maps.Keys(ignored)) {
		val.addDiagnostic(diagnostic.Warning(ignored[name], fmt.Sprintf(
			"Job <local>/%s: '%s' is not a recognized approval key and is ignored", jobInvocation.JobName, name)))
	}
}

// Validates the structure of an invocation of a job-group, which
// does not have all of the same features as a single job invocation.
func (val Validate) validateJobGroupInvocation(jobInvocation ast2.JobInvocation, ctx InvocationContext) {
	if ctx.Kind == InJobGroup {
		val.addDiagnostic(diagnostic.Error(jobInvocation.JobNameRange,
			fmt.Sprintf("Job group \"%s\" cannot reference job group \"%s\" -- nesting is not supported", ctx.JobGroupName, jobInvocation.JobName)))
		return // exit early
	}

	// Keys not allowed in job group invocations
	if jobInvocation.HasMatrix {
		val.addDiagnostic(diagnostic.Error(jobInvocation.MatrixRange, "Job group invocations do not support `matrix`"))
	}
	if jobInvocation.OverrideWith != "" {
		val.addDiagnostic(diagnostic.Error(jobInvocation.OverrideWithRange, "Job group invocations do not support use of `override-with`"))
	}
	if jobInvocation.Type != "" {
		val.addDiagnostic(diagnostic.Error(jobInvocation.TypeRange, "Job group invocations do not support use of `type`"))
	}
	if len(jobInvocation.Parameters) > 0 {
		paramNames := make([]string, 0, len(jobInvocation.Parameters))
		for name := range jobInvocation.Parameters {
			paramNames = append(paramNames, fmt.Sprintf("`%s`", name))
		}
		sort.Strings(paramNames) // Since map iteration is not guaranteed to be in order, sort the paramNames
		val.addDiagnostic(diagnostic.Error(jobInvocation.JobInvocationRange,
			fmt.Sprintf("Job group invocations do not support custom parameters, but found: %s", strings.Join(paramNames, ", "))))
	}
	if len(jobInvocation.Context) > 0 {
		val.addDiagnostic(diagnostic.Error(jobInvocation.JobInvocationRange, "Job group invocations do not support use of `context`"))
	}
	if len(jobInvocation.PreSteps) > 0 {
		val.addDiagnostic(diagnostic.Error(jobInvocation.PreStepsRange, "Job group invocations do not support use of `pre-steps`"))
	}
	if len(jobInvocation.PostSteps) > 0 {
		val.addDiagnostic(diagnostic.Error(jobInvocation.PostStepsRange, "Job group invocations do not support use of `post-steps`"))
	}
}

func (val Validate) validateDAG(invocations []ast2.JobInvocation, dag map[string][]string) {
	nodesInCycle := isValidDAG(dag)

	for _, node := range nodesInCycle {
		for _, invocation := range invocations {
			if invocation.JobName == node {
				val.addDiagnostic(diagnostic.Error(
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
