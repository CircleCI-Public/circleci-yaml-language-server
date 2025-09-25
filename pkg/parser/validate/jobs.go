package validate

import (
	"fmt"
	"slices"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func (val Validate) ValidateJobs() {
	for _, job := range val.Doc.Jobs {
		val.validateSingleJob(job)
	}
}

func (val Validate) validateSingleJob(job ast.Job) {
	val.validateJobType(job)

	val.validateSteps(job.Steps, job.Name, job.Parameters)

	if job.Steps != nil && (job.Type == "approval" || job.Type == "no-op" || job.Type == "release") {
		val.addDiagnostic(
			protocol.Diagnostic{
				Range:    job.StepsRange,
				Message:  "If job type is approval, no-op or release, then steps will be ignored.",
				Severity: protocol.DiagnosticSeverityWarning,
			},
		)
	}

	// Local orbs do not need unused checks because those checks collides with the overall YAML unused checks
	if !val.IsLocalOrb && !val.checkIfJobIsUsed(job) {
		val.jobIsUnused(job)
	}

	if !utils.HasStoreTestResultStep(job.Steps) && strings.Contains(job.Name, "test") {
		val.addDiagnostic(
			protocol.Diagnostic{
				Range:    job.NameRange,
				Message:  "You may want to add the `store_test_results` step to visualize the test results in CircleCI",
				Severity: protocol.DiagnosticSeverityHint,
			},
		)
	}

	if job.Executor != "" {
		if utils.CheckIfOnlyParamUsed(job.Executor) {
			_, paramName := utils.ExtractParameterName(job.Executor)
			param := job.Parameters[paramName]

			checkParam := func(executorDefault string, rng protocol.Range) {
				isOrbExecutor, err := val.doesOrbExecutorExist(executorDefault, rng)
				if !param.IsOptional() {
					val.addDiagnostic(
						protocol.Diagnostic{
							Range: rng,
							Message: fmt.Sprintf(
								"No default value specified for parameter `%s`.",
								paramName,
							),
							Severity: protocol.DiagnosticSeverityWarning,
						},
					)
				} else if val.Context.Api.UseDefaultInstance() && !val.Doc.DoesExecutorExist(executorDefault) &&
					(!isOrbExecutor && err == nil) {
					// Error on the default value
					val.addDiagnostic(
						protocol.Diagnostic{
							Range: rng,
							Message: fmt.Sprintf(
								"Parameter is used as executor but executor `%s` does not exist.",
								executorDefault,
							),
							Severity: protocol.DiagnosticSeverityError,
						},
					)
				}
			}

			if param != nil {
				switch param := param.(type) {
				case ast.StringParameter:
				case ast.ExecutorParameter:
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
		}
	}

	// By default Parallelism is set to -1; see parser.parseSingleJob
	if job.Parallelism == 0 || job.Parallelism == 1 {
		val.addDiagnostic(
			protocol.Diagnostic{
				Range:    job.ParallelismRange,
				Message:  "To benefit from parallelism, you should select a value greater than 1. You can read more about how to leverage parallelism to speed up pipelines in the CircleCI docs.",
				Severity: protocol.DiagnosticSeverityWarning,
				CodeDescription: &protocol.CodeDescription{
					Href: "https://circleci.com/docs/parallelism-faster-jobs/",
				},
				Source: "More info",
				Code:   "Docs",
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
	}
}

func (val Validate) checkIfJobIsUsed(job ast.Job) bool {
	for _, definedJob := range val.Doc.Jobs {
		if val.checkIfStepsContainStep(definedJob.Steps, job.Name) {
			return true
		}
	}

	for _, workflow := range val.Doc.Workflows {
		for _, jobRef := range workflow.JobRefs {
			if jobRef.JobName == job.Name {
				return true
			}
		}
	}

	return false
}

func (val Validate) jobIsUnused(job ast.Job) {
	val.addDiagnostic(utils.CreateWarningDiagnosticFromRange(job.NameRange, "Job is unused"))
}

func (val Validate) validateJobType(job ast.Job) {
	// Default job type is build, therefore empty `type:` is valid. No need to validate further
	if job.Type == "" {
		return
	}

	if !slices.Contains(utils.JobTypes, job.Type) {
		val.addDiagnostic(
			utils.CreateErrorDiagnosticFromRange(
				job.TypeRange,
				fmt.Sprintf("Invalid job type '%s'. Allowed types: %s",
					job.Type,
					strings.Join(utils.JobTypes, ", "))))

		return
	}

	if job.Type == "build" {
		val.addDiagnostic(
			protocol.Diagnostic{
				Range:    job.TypeRange,
				Message:  "If no `type:` key is specified, the job will default to `type: build`.",
				Severity: protocol.DiagnosticSeverityHint,
			},
		)
	}
}
