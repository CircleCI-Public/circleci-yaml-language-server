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
	val.validateSteps(job.Steps, job.Name)

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

			if param != nil {
				switch param.(type) {
				case ast.StringParameter:
					str := param.(ast.StringParameter)

					if str.IsOptional() && !val.Doc.DoesExecutorExist(str.Default) {
						// Error on the default value
						val.addDiagnostic(
							protocol.Diagnostic{
								Range: str.DefaultRange,
								Message: fmt.Sprintf(
									"Parameter is used as executor but executor `%s` does not exist.",
									str.Default,
								),
								Severity: protocol.DiagnosticSeverityError,
							},
						)
					}
				case ast.ExecutorParameter:
					exec := param.(ast.ExecutorParameter)

					if exec.IsOptional() && !val.Doc.DoesExecutorExist(exec.Default) {
						// Error on the default value
						val.addDiagnostic(
							protocol.Diagnostic{
								Range: exec.DefaultRange,
								Message: fmt.Sprintf(
									"Parameter is used as executor but executor `%s` does not exist.",
									exec.Default,
								),
								Severity: protocol.DiagnosticSeverityError,
							},
						)
					}
				}

			}

		} else if !val.Doc.DoesExecutorExist(job.Executor) {
			if val.Doc.IsOrb(job.Executor) {
				val.validateOrbExecutor(job.Executor, job.ExecutorRange)
			} else {
				val.addDiagnostic(
					protocol.Diagnostic{
						Range:    job.ExecutorRange,
						Message:  "Executor `" + job.Executor + "` does not exist",
						Severity: protocol.DiagnosticSeverityError,
					},
				)
			}
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

	for _, img := range job.Docker.Image {
		isValid, errMessage := ValidateDockerImage(&img, &val.Cache.DockerCache)

		if !isValid {
			val.addDiagnostic(
				utils.CreateErrorDiagnosticFromRange(
					img.ImageRange,
					errMessage,
				),
			)
		}
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
