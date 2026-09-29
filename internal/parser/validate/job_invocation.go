package validate

import (
	"fmt"
	"maps"
	"regexp"
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

	// WorkflowName is the name of the enclosing workflow.
	// Only meaningful when Kind == InWorkflow; empty otherwise.
	WorkflowName string
}

func (val Validate) doesJobInvocationExist(jobInvocations []ast2.JobInvocation, requireName string) bool {
	for _, jobInvocation := range jobInvocations {
		names := append([]string{jobInvocation.JobName, jobInvocation.StepName, jobInvocation.MatrixAlias}, jobInvocation.MatrixNames...)
		if slices.ContainsFunc(names, func(name string) bool {
			// A matrix's `name:` template is only a name once MatrixNames expand it.
			return !paramref.IsMatrixPartiallyReferenced(name) && invocationNameMatches(name, requireName)
		}) {
			return true
		}
	}
	return false
}

// validateRequireIsUnambiguous reports a require of the invocation at index
// that names more than one of the other invocations, as jobs or matrices.
func (val Validate) validateRequireIsUnambiguous(jobInvocations []ast2.JobInvocation, index int, require ast2.Require) {
	jobs, matrices := 0, 0
	for i, other := range jobInvocations {
		switch {
		case i == index:
		case other.HasMatrix && other.MatrixAlias == require.Name:
			matrices++
		case !other.HasMatrix && other.StepName == require.Name:
			jobs++
		}
	}
	if jobs+matrices < 2 {
		return
	}

	named := counted(jobs, "other job", "other jobs")
	if matrices > 0 {
		named += " and " + counted(matrices, "matrix", "matrices")
	}
	val.addDiagnostic(diagnostic.Error(require.Range, fmt.Sprintf(
		"Job '%s' requires '%s', which is the name of %s in this workflow. "+
			"Give each of them a unique `name`, or a matrix a unique `alias`, to require one",
		jobInvocations[index].StepName, require.Name, named)))
}

func counted(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

// invocationNameMatches reports whether a job's name is the name a `requires`
// gives. A name holding a reference, such as `deploy-<< pipeline.git.branch >>`,
// is only known once the pipeline runs, and so is a require holding one.
func invocationNameMatches(name, requireName string) bool {
	nameIsTemplate, requireIsTemplate := paramref.ContainsReference(name), paramref.ContainsReference(requireName)
	switch {
	case nameIsTemplate && requireIsTemplate:
		return paramref.CouldBothExpandTo(name, requireName)
	case nameIsTemplate:
		return paramref.CouldExpandTo(name, requireName)
	case requireIsTemplate:
		return paramref.CouldExpandTo(requireName, name)
	}
	return name == requireName
}

// validateMatrixRequireExists checks a require holding `<< matrix.x >>`
// against the other invocations once for each job the matrix expands to, as
// the compiler expands it. Outside a matrix, it can't be expanded.
func (val Validate) validateMatrixRequireExists(jobInvocations []ast2.JobInvocation, jobInvocation ast2.JobInvocation, require ast2.Require, ctx InvocationContext) {
	for i, combination := range jobInvocation.MatrixCombinations {
		expanded := parser.ExpandMatrixReferences(require.Name, combination)
		if paramref.IsMatrixPartiallyReferenced(expanded) || val.doesJobInvocationExist(jobInvocations, expanded) {
			continue
		}
		missing := "a job in this workflow"
		if ctx.Kind == InJobGroup {
			missing = fmt.Sprintf("a member of the job group '%s'", ctx.JobGroupName)
		}
		val.addDiagnostic(diagnostic.Error(require.Range, fmt.Sprintf(
			"Job '%s' requires '%s', which is not %s", jobInvocation.MatrixNames[i], expanded, missing)))
	}
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
			if definedParam.GetType() == "executor" {
				val.validateExecutorArgumentReference(jobInvocation.Parameters[definedParam.GetName()], definedParams)
			}
		}
	}

	for _, param := range jobInvocation.Parameters {
		if definedParams[param.Name] == nil {
			val.addDiagnostic(diagnostic.Error(
				param.Range,
				fmt.Sprintf("Parameter %s is not defined for %s", param.Name, jobName)),
			)
		}
	}

	for name, values := range jobInvocation.MatrixParams {
		if definedParams[name] == nil && len(values) > 0 {
			val.addDiagnostic(diagnostic.Error(
				values[0].Range,
				fmt.Sprintf("Parameter %s is not defined for %s", name, jobName)),
			)
		}
	}
}

// hasBeenRenamed returns true when the invocation has an explicit name: attribute
// (as opposed to the parser's default of StepName == JobName).
func hasBeenRenamed(inv ast2.JobInvocation) bool {
	return inv.StepName != inv.JobName
}

// validateDuplicateNames reports invocations that share a name: job groups
// in a workflow, or jobs in a job group. An invocation without a `name` is
// named after what it invokes.
func (val Validate) validateDuplicateNames(jobInvocations []ast2.JobInvocation, ctx InvocationContext) {
	byName := map[string][]ast2.JobInvocation{}
	var names []string
	for _, inv := range jobInvocations {
		isJobGroup := val.Doc.DoesJobGroupExist(inv.JobName)
		if inv.HasMatrix || (ctx.Kind == InWorkflow) != isJobGroup {
			continue
		}
		if _, ok := byName[inv.StepName]; !ok {
			names = append(names, inv.StepName)
		}
		byName[inv.StepName] = append(byName[inv.StepName], inv)
	}

	for _, name := range names {
		invocations := byName[name]
		if len(invocations) < 2 {
			continue
		}
		message := fmt.Sprintf("Job-group '%s' occurs %d times in workflow '%s'. "+
			"You can give a job-group an explicit name by adding a `name` key",
			name, len(invocations), ctx.WorkflowName)
		if ctx.Kind == InJobGroup {
			message = fmt.Sprintf("Job '%s' occurs %d times in job group '%s'. "+
				"You can give a job within a job group an explicit name by adding a `name` key",
				name, len(invocations), ctx.JobGroupName)
		}
		for _, inv := range invocations {
			rng := inv.JobNameRange
			if hasBeenRenamed(inv) {
				rng = inv.StepNameRange
			}
			val.addDiagnostic(diagnostic.Error(rng, message))
		}
	}
}

// Validates and adds diagnostics for workflow/job-group job invocations.
func (val Validate) validateInvocations(jobInvocations []ast2.JobInvocation, ctx InvocationContext) {
	val.validateDuplicateNames(jobInvocations, ctx)
	for i, jobInvocation := range jobInvocations {
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
			val.validateRequireIsUnambiguous(jobInvocations, i, require)

			if paramref.IsMatrixPartiallyReferenced(require.Name) {
				val.validateMatrixRequireExists(jobInvocations, jobInvocation, require, ctx)
			} else if !val.doesJobInvocationExist(jobInvocations, require.Name) {
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
		val.validateApprovalInvocation(jobInvocation, ctx)
		val.validateInvocationContexts(jobInvocation)
		return
	}

	// This orb check is not needed for job-groups because we don't support job-groups in orbs.
	if val.Doc.IsFromUnfetchableOrb(jobInvocation.JobName, val.Cache) {
		return
	}

	if message := val.unknownJobMessage(jobInvocation.JobName); message != "" {
		val.addDiagnostic(diagnostic.Error(jobInvocation.JobInvocationRange, message))
		return
	}

	// An alias that names nothing is reported where it is declared.
	alias, isAlias := val.Doc.JobAlias(jobInvocation.JobName)
	if !val.Doc.IsBuiltIn(jobInvocation.JobName) && (!isAlias || val.jobAliasProblem(alias) == "") {
		if target, ok := val.overrideTarget(jobInvocation); ok {
			invoked := jobInvocation
			invoked.JobName = target
			val.validateJobInvocationParameters(invoked)
		}
	}

	val.validateInvocationContexts(jobInvocation)
}

// overrideTarget returns the job an invocation runs, which is its
// `override-with` job when the orb has it, and otherwise the job it names.
// It is false when the job's parameters can't be known.
func (val Validate) overrideTarget(jobInvocation ast2.JobInvocation) (string, bool) {
	target := jobInvocation.OverrideWith
	if target == "" || paramref.ContainsReference(target) {
		return jobInvocation.JobName, true
	}

	orbName, jobName, isOrbReference := strings.Cut(target, "/")
	if !isOrbReference {
		val.addDiagnostic(diagnostic.Error(jobInvocation.OverrideWithRange,
			fmt.Sprintf("override-with: \"%s\" does not reference a job definition in an orb", target)))
		return "", false
	}

	if _, ok := val.Doc.Orbs[orbName]; !ok {
		val.addDiagnostic(diagnostic.Warning(jobInvocation.OverrideWithRange, fmt.Sprintf(
			"override-with: orb %s is not declared, so this runs the job %s", orbName, jobInvocation.JobName)))
		return jobInvocation.JobName, true
	}

	if val.Doc.IsFromUnfetchableOrb(target, val.Cache) {
		return "", false
	}

	orbInfo, err := val.Doc.GetOrbInfoFromName(orbName, val.Cache)
	if err != nil || orbInfo == nil {
		return "", false
	}
	if _, ok := orbInfo.Jobs[jobName]; ok {
		return target, true
	}

	val.addDiagnostic(diagnostic.Warning(jobInvocation.OverrideWithRange, fmt.Sprintf(
		"override-with: orb %s has no job %s, so this runs the job %s", orbName, jobName, jobInvocation.JobName)))
	return jobInvocation.JobName, true
}

func (val Validate) isKnownJob(name string) bool {
	_, isJobAlias := val.Doc.JobAlias(name)
	return val.Doc.DoesJobExist(name) || isJobAlias || val.Doc.IsOrbJob(name, val.Cache)
}

func (val Validate) isCommand(name string) bool {
	_, isCommandAlias := val.Doc.CommandAlias(name)
	return val.Doc.DoesCommandExist(name) || isCommandAlias || val.Doc.IsOrbCommand(name, val.Cache)
}

func (val Validate) unknownJobMessage(name string) string {
	switch {
	case val.isKnownJob(name):
		return ""
	case val.isCommand(name):
		return fmt.Sprintf("%s is a command, not a job: a workflow runs jobs", name)
	default:
		return fmt.Sprintf("Cannot find declaration for job \"%s\"", name)
	}
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
// jobNamePattern is the shape of a job's name under `jobs:`. An approval job
// is named in the workflow, where nothing checks that it has this shape.
var jobNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z\s\d_-]*$`)

func (val Validate) validateApprovalInvocation(jobInvocation ast2.JobInvocation, ctx InvocationContext) {
	if !jobNamePattern.MatchString(jobInvocation.JobName) && !paramref.ContainsReference(jobInvocation.JobName) {
		val.addDiagnostic(diagnostic.Warning(jobInvocation.JobNameRange, fmt.Sprintf(
			"Approval job '%s' is not a valid job name: it must start with a letter and contain only "+
				"letters, digits, whitespace, underscores and hyphens", jobInvocation.JobName)))
	}

	switch {
	case !val.Doc.DoesJobExist(jobInvocation.JobName):
	case ctx.Kind == InJobGroup:
		val.addDiagnostic(diagnostic.Error(jobInvocation.JobNameRange, fmt.Sprintf(
			"Duplicate job definition: '%s' is defined in `jobs:`, so a job group can't also define it "+
				"with type: approval", jobInvocation.JobName)))
	default:
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
