package validate

import (
	"fmt"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

var WHEN_KEYWORDS = []string{
	"on_success",
	"always",
	"on_fail",
}

func (val Validate) validateSteps(steps []ast.Step, name string, jobOrCommandParameters map[string]ast.Parameter) error {
	for _, step := range steps {
		switch step := step.(type) {
		case ast.Run:
			val.validateRunCommand(step, jobOrCommandParameters)
		case ast.NamedStep:
			val.validateNamedStep(step, jobOrCommandParameters)
		case ast.Steps:
			val.validateStepSteps(step, name)
		}
	}
	return nil
}

func (val Validate) validateRunCommand(step ast.Run, jobOrCommandParameters map[string]ast.Parameter) {
	val.shellCheck(step)
	val.validateRunCommandWhenField(step, jobOrCommandParameters)
}

func (val Validate) validateRunCommandWhenField(step ast.Run, jobOrCommandParameters map[string]ast.Parameter) {
	var value string
	if step.When == "" && step.WhenRange.Start.Line == 0 {
		return
	}

	// If the when field is a parameter, such as:
	// when: << parameters.my_param >>
	if utils.CheckIfOnlyParamUsed(step.When) {
		paramName, isPipelineParam := utils.GetParamNameUsedAtPos(val.Doc.Content, step.WhenRange.End)
		var param ast.Parameter
		var ok bool

		if isPipelineParam {
			param, ok = val.Doc.PipelinesParameters[paramName]
		} else {
			param, ok = jobOrCommandParameters[paramName]
		}

		if !ok {
			return
		}

		if param.IsOptional() {
			switch param := param.(type) {
			case ast.StringParameter:
				value = param.Default
			default:
				val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
					step.WhenRange,
					fmt.Sprintf("Parameter %s is not a string type parameter, and therefore cannot be used inside the `when` field", paramName),
				))
				return
			}
		}
	} else {
		value = step.When
	}
	if utils.FindInArray(WHEN_KEYWORDS, value) < 0 {
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
			step.WhenRange,
			fmt.Sprintf("Invalid when condition: expected `%s`; got `%s`", strings.Join(WHEN_KEYWORDS, "`, `"), value)))
	}
}

func (val Validate) validateNamedStep(step ast.NamedStep, usableParams map[string]ast.Parameter) {
	commandExists := val.Doc.DoesJobExist(step.Name) ||
		val.Doc.DoesCommandExist(step.Name) ||
		val.Doc.IsBuiltIn(step.Name) ||
		val.Doc.IsOrbCommand(step.Name, val.Cache) ||
		val.Doc.IsAlias(step.Name)

	if !commandExists {
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
			step.Range,
			fmt.Sprintf("Cannot find declaration for step %s", step.Name)))
	}

	if !val.Doc.IsBuiltIn(step.Name) {
		targetEntityDefinedParams := val.Doc.GetDefinedParams(step.Name, val.Cache)
		val.validateParametersValue(
			step.Parameters,
			step.Name,
			step.Range,
			targetEntityDefinedParams,
			usableParams,
		)
	}

	if step.Name == "store_test_results" {
		val.addDiagnostic(
			protocol.Diagnostic{
				Message:  "Path must be specified for `store_test_results` step",
				Range:    step.Range,
				Severity: protocol.DiagnosticSeverityError,
			})
	}
}

func (val Validate) validateStepSteps(step ast.Steps, name string) {
	if !val.Doc.DoesCommandExist(name) {
		return
	}
	command := val.Doc.Commands[name]
	parameter, ok := command.Parameters[step.Name]
	if !ok {
		return
	}
	parameterType := parameter.GetType()
	if parameterType != "steps" {
		val.addDiagnostic(protocol.Diagnostic{
			Severity: protocol.DiagnosticSeverityError,
			Range:    step.Range,
			Message:  "Parameter type is not steps",
			Source:   "cci-language-server",
		})
	}
}

func (val Validate) checkIfStepsContainStep(steps []ast.Step, stepName string) bool {
	for _, step := range steps {
		if step.GetName() == stepName {
			return true
		}
	}

	return false
}

func (val Validate) checkIfStepsContainOrb(steps []ast.Step, orbName string) bool {
	for _, step := range steps {
		isOrb := val.Doc.IsOrbReference(step.GetName())

		if isOrb && strings.Split(step.GetName(), "/")[0] == orbName {
			return true
		}
	}

	return false
}

func (val Validate) checkIfJobUseOrb(job ast.Job, orbName string) bool {
	if val.checkIfStepsContainOrb(job.Steps, orbName) {
		return true
	}

	if job.Executor != "" {
		split := strings.Split(job.Executor, "/")
		if split[0] == orbName {
			return true
		}
	}

	return false
}
