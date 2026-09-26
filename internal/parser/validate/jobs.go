package validate

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"go.lsp.dev/protocol"

	ast2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/paramref"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func (val Validate) ValidateJobs() {
	for _, job := range val.Doc.Jobs {
		val.validateSingleJob(job)
	}
}

func (val Validate) validateSingleJob(job ast2.Job) {
	val.validateJobType(job)
	val.validateReservedParameterNames(job)

	val.validateSteps(job.Steps, job.Name, job.Parameters)
	val.validateRemoteDockerOnce(job)

	if job.Steps != nil && job.Type != "" && job.Type != "build" {
		val.addDiagnostic(
			protocol.Diagnostic{
				Range:    job.StepsRange,
				Message:  protocol.String("Steps only exist in `build` jobs. Steps here will be ignored."),
				Severity: protocol.DiagnosticSeverityWarning,
			},
		)
	}

	// Local orbs do not need unused checks because those checks collides with the overall YAML unused checks
	if !val.IsLocalOrb {
		val.checkAndReportUnusedJob(job)
	}

	if !ast2.HasStoreTestResultStep(job.Steps) && strings.Contains(job.Name, "test") {
		val.addDiagnostic(
			protocol.Diagnostic{
				Range:    job.NameRange,
				Message:  protocol.String("You may want to add the `store_test_results` step to visualize the test results in CircleCI"),
				Severity: protocol.DiagnosticSeverityHint,
			},
		)
	}

	if job.Executor != "" {
		if paramref.IsOnlyParameter(job.Executor) {
			_, paramName := paramref.ExtractName(job.Executor)
			param := job.Parameters[paramName]

			checkParam := func(executorDefault string, rng protocol.Range) {
				// A parameter without a default is required, which is checked
				// where the job is invoked.
				if !param.IsOptional() {
					return
				}
				isOrbExecutor, err := val.doesOrbExecutorExist(executorDefault, rng)
				if val.Context.Api.UseDefaultInstance() && !val.Doc.DoesExecutorExist(executorDefault) &&
					(!isOrbExecutor && err == nil) {
					// Error on the default value
					val.addDiagnostic(
						protocol.Diagnostic{
							Range: rng,
							Message: protocol.String(fmt.Sprintf(
								"Parameter is used as executor but executor `%s` does not exist.",
								executorDefault,
							)),
							Severity: protocol.DiagnosticSeverityError,
						},
					)
				}
			}

			if param != nil {
				switch param := param.(type) {
				case ast2.StringParameter:
				case ast2.ExecutorParameter:
					checkParam(param.Default, job.ExecutorRange)
				}
			}

		} else if !val.Doc.DoesExecutorExist(job.Executor) {
			val.validateExecutorReference(job.Executor, job.ExecutorRange)
		} else {
			executor := val.Doc.Executors[job.Executor]
			val.validateParametersValue(
				job.ExecutorParameters,
				executor.GetName(),
				job.ExecutorRange,
				executor.GetParameters(),
				job.Parameters,
			)
			val.validateExecutorOverrides(job, executor)
			val.validateExecutorHasType(job, executor)
		}
	}

	// By default Parallelism is set to -1; see parser.parseSingleJob
	if job.Parallelism == 1 {
		val.addDiagnostic(
			protocol.Diagnostic{
				Range:    job.ParallelismRange,
				Message:  protocol.String("To benefit from parallelism, you should select a value greater than 1. You can read more about how to leverage parallelism to speed up pipelines in the CircleCI docs."),
				Severity: protocol.DiagnosticSeverityWarning,
				CodeDescription: protocol.CodeDescription{
					Href: "https://circleci.com/docs/parallelism-faster-jobs/",
				},
				Source: protocol.NewOptional("More info"),
				Code:   protocol.String("Docs"),
			},
		)
	}

	if job.Retention.Caches.Text != "" {
		val.validateRetention(job.Retention)
	}

	if len(job.Docker.Image) > 0 {
		val.validateDockerExecutor(job.Docker)
	} else if job.MacOS.Xcode != "" {
		val.validateMacOSExecutor(job.MacOS)
	} else if job.Machine.Image != "" {
		val.validateMachineExecutor(job.Machine)
	} else {
		// Such as `machine: true` on a self-hosted runner, which the executor
		// checks don't see.
		val.validateRunnerResourceClass(job.ResourceClass, job.ResourceClassRange)
	}
}

// validateRemoteDockerOnce reports each setup_remote_docker after a job's
// first, which the compiler rejects.
// reservedJobParameters are the keys a workflow gives a job invocation
// itself, so a job can't take a parameter of the same name.
var reservedJobParameters = []string{
	"name", "pre-steps", "post-steps", "filters", "requires", "context", "type", "override-with", "upstream",
}

func (val Validate) validateReservedParameterNames(job ast2.Job) {
	quoted := make([]string, len(reservedJobParameters))
	for i, name := range reservedJobParameters {
		quoted[i] = fmt.Sprintf("%q", name)
	}
	last := len(quoted) - 1
	message := strings.Join(quoted[:last], ", ") + ", and " + quoted[last] + " are reserved parameter names in build jobs"

	for _, name := range reservedJobParameters {
		if param, ok := job.Parameters[name]; ok {
			val.addDiagnostic(diagnostic.Error(param.GetNameRange(), message))
		}
	}
}

func (val Validate) validateRemoteDockerOnce(job ast2.Job) {
	seen := false
	for _, step := range job.Steps {
		isRemoteDocker := false
		switch step := step.(type) {
		case ast2.SetupRemoteDocker:
			isRemoteDocker = true
		case ast2.NamedStep:
			isRemoteDocker = step.Name == "setup_remote_docker"
		}
		if !isRemoteDocker {
			continue
		}
		if seen {
			val.addDiagnostic(diagnostic.Error(step.GetRange(), fmt.Sprintf(
				"More than one setup_remote_docker is not valid, please adjust in job %s.", job.Name)))
		}
		seen = true
	}
}

// validateExecutorOverrides warns about a setting given both on the job and on
// its executor, where the job's silently wins.
// validateExecutorHasType checks that a job using an executor with no
// docker, machine or macos key gives one itself.
func (val Validate) validateExecutorHasType(job ast2.Job, executor ast2.Executor) {
	if _, typeless := executor.(ast2.BaseExecutor); !typeless {
		return
	}
	if !position.IsDefaultRange(job.DockerRange) || !position.IsDefaultRange(job.MachineRange) ||
		!position.IsDefaultRange(job.MacOSRange) {
		return
	}
	val.addDiagnostic(diagnostic.Error(job.ExecutorRange, fmt.Sprintf(
		`Executor %s is missing a required key: "docker", "machine", or "macos"`, executor.GetName())))
}

func (val Validate) validateExecutorOverrides(job ast2.Job, executor ast2.Executor) {
	if job.ResourceClass != "" && executor.GetResourceClass() != "" {
		val.addDiagnostic(diagnostic.Warning(job.ResourceClassRange,
			"resource_class is set both on the job and on the executor; the job's "+
				"value is used and the executor's is ignored. See "+
				"https://circleci.com/docs/reference/configuration-reference/#executors"))
	}

	if job.Shell != "" && executor.GetShell() != "" {
		val.addDiagnostic(diagnostic.Warning(job.ShellRange,
			"shell is set both on the job and on the executor; the job's value is "+
				"used and the executor's is ignored. See "+
				"https://circleci.com/docs/reference/configuration-reference/#executors"))
	}
}

func (val Validate) checkAndReportUnusedJob(job ast2.Job) {
	if job.Name == "build" && val.Doc.HasNoWorkflows() {
		return
	}

	// Used directly in another job's steps
	for _, definedJob := range val.Doc.Jobs {
		if val.checkIfStepsContainStep(definedJob.Steps, job.Name) {
			return
		}
	}

	// Used directly in a workflow
	for _, workflow := range val.Doc.Workflows {
		for _, jobInvocation := range workflow.JobInvocations {
			if jobInvocation.JobName == job.Name {
				return
			}
		}
	}

	// Collect all job-groups that contain this job
	var unusedGroups []string
	for groupName, group := range val.Doc.JobGroups {
		for _, jobInvocation := range group.JobInvocations {
			// We compare against JobName (the original definition name), not StepName,
			// because StepName is just a user-chosen alias for the invocation - the
			// underlying job being referenced is always identified by JobName.
			if jobInvocation.JobName == job.Name {
				if val.isJobGroupUsedInWorkflows(groupName) {
					// At least one group containing this job is used — job counts as used
					return
				}
				unusedGroups = append(unusedGroups, groupName)
			}
		}
	}

	if len(unusedGroups) > 0 {
		sort.Strings(unusedGroups)
		val.addDiagnostic(diagnostic.Warning(
			job.NameRange,
			fmt.Sprintf("Job \"%s\" is used in job group \"%s\", but that group is never invoked in a workflow", job.Name, unusedGroups[0]),
		))
		return
	}

	// Not referenced anywhere
	val.addDiagnostic(diagnostic.Warning(job.NameRange, "Job is unused"))
}

// isJobGroupUsedInWorkflows returns true if any workflow references the given
// job-group name (either directly as JobName or via StepName).
// We match on JobName because it identifies the actual entity being invoked;
// StepName is only a display alias and doesn't change which job/group is referenced.
func (val Validate) isJobGroupUsedInWorkflows(groupName string) bool {
	for _, workflow := range val.Doc.Workflows {
		for _, jobInvocation := range workflow.JobInvocations {
			if jobInvocation.JobName == groupName {
				return true
			}
		}
	}
	return false
}

func (val Validate) validateJobType(job ast2.Job) {
	// Default job type is build, therefore empty `type:` is valid. No need to validate further
	if job.Type == "" {
		return
	}

	if !slices.Contains(ast2.JobTypes, job.Type) {
		val.addDiagnostic(
			diagnostic.Error(
				job.TypeRange,
				fmt.Sprintf("Invalid job type '%s'. Allowed types: %s",
					job.Type,
					strings.Join(ast2.JobTypes, ", "))))

		return
	}

	if job.Type == "build" {
		val.addDiagnostic(
			protocol.Diagnostic{
				Range:    job.TypeRange,
				Message:  protocol.String("If no `type:` key is specified, the job will default to `type: build`."),
				Severity: protocol.DiagnosticSeverityHint,
			},
		)
	}
}
