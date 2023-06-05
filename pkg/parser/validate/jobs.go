package validate

import (
	"fmt"
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
	val.validateSteps(job.Steps, job.Name, job.Parameters)

	if !val.checkIfJobIsUsed(job) {
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
				if val.Context.Api.UseDefaultInstance() && !val.Doc.DoesExecutorExist(executorDefault) &&
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

	val.validateDockerExecutor(job.Docker)
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
