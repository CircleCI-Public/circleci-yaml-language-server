package validate

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

var WHEN_KEYWORDS = []string{
	"on_success",
	"always",
	"on_fail",
}

// AutoRerunDelay validation regex: matches 1-10 minutes or any number of seconds (but not both)
var AUTO_RERUN_DELAY_REGEX = regexp.MustCompile(`^((10|[1-9])m|([1-9][0-9]*)s)$`)

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
	if step.IsDeployStep {
		val.addDiagnostic(protocol.Diagnostic{
			Range:    step.Range,
			Message:  "The `deploy` step is deprecated. Please use the `run` job instead.",
			Severity: protocol.DiagnosticSeverityWarning,
			Tags: []protocol.DiagnosticTag{
				protocol.DiagnosticTagDeprecated,
			},
		})
	}

	// Validate that background steps cannot use max_auto_reruns or auto_rerun_delay
	if step.Background && (step.MaxAutoReruns != "" || step.AutoRerunDelay != "") {
		val.addDiagnostic(protocol.Diagnostic{
			Range:    step.Range,
			Message:  "Background steps cannot use max_auto_reruns or auto_rerun_delay fields",
			Severity: protocol.DiagnosticSeverityError,
		})
	}

	// Validate that auto_rerun_delay requires max_auto_reruns
	if step.AutoRerunDelay != "" && step.MaxAutoReruns == "" {
		val.addDiagnostic(protocol.Diagnostic{
			Range:    step.Range,
			Message:  "auto_rerun_delay requires max_auto_reruns to be specified",
			Severity: protocol.DiagnosticSeverityError,
		})
	}

	// Validate that max_auto_reruns is between 1 and 5
	if step.MaxAutoReruns != "" {
		rerunCount, err := strconv.Atoi(step.MaxAutoReruns)
		if err != nil || rerunCount <= 0 || rerunCount > 5 {
			val.addDiagnostic(protocol.Diagnostic{
				Range:    step.Range,
				Message:  "max_auto_reruns must be between 1 and 5",
				Severity: protocol.DiagnosticSeverityError,
			})
		}
	}

	// Validate that auto_rerun_delay conforms to the specific format and duration limits
	if step.AutoRerunDelay != "" {
		// First check if it's a valid duration
		duration, err := time.ParseDuration(step.AutoRerunDelay)
		if err != nil {
			val.addDiagnostic(protocol.Diagnostic{
				Range:    step.Range,
				Message:  "auto_rerun_delay must be a valid duration",
				Severity: protocol.DiagnosticSeverityError,
			})
		} else {
			// Check if it matches the required format
			if !AUTO_RERUN_DELAY_REGEX.MatchString(step.AutoRerunDelay) {
				val.addDiagnostic(protocol.Diagnostic{
					Range:    step.Range,
					Message:  "auto_rerun_delay must be in the format of 1-10 minutes (e.g., '1m', '10m') or any number of seconds (e.g., '30s', '120s')",
					Severity: protocol.DiagnosticSeverityError,
				})
			}
			// Check if duration exceeds 10 minutes
			if duration > 10*time.Minute {
				val.addDiagnostic(protocol.Diagnostic{
					Range:    step.Range,
					Message:  "auto_rerun_delay must not exceed 10 minutes",
					Severity: protocol.DiagnosticSeverityError,
				})
			}
		}
	}

	var value string
	// If the when field is a parameter, such as:
	// when: << parameters.my_param >>
	if step.When == "" && step.WhenRange.Start.Line == 0 {
		return
	}

	if utils.CheckIfOnlyParamUsed(step.When) {
		paramName, isPipelineParam := utils.GetParamNameUsedAtPos(val.Doc.Content, step.WhenRange.End)
		var param ast.Parameter
		var ok bool

		if isPipelineParam {
			param, ok = val.Doc.PipelineParameters[paramName]
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

	if !slices.Contains(WHEN_KEYWORDS, value) {
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

	if val.Doc.IsFromUnfetchableOrb(step.Name) {
		return
	}

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

func (val Validate) checkIfJobParamContainOrb(params map[string]ast.ParameterValue, orbName string) bool {
	for _, p := range params {
		array, ok := p.Value.([]ast.ParameterValue)
		if !ok {
			continue
		}

		for _, value := range array {
			if value.Type != "steps" {
				break
			}

			steps, ok := value.Value.([]ast.Step)
			if !ok {
				continue
			}

			for _, step := range steps {
				name := step.GetName()
				split := strings.Split(name, "/")
				if len(split) != 2 {
					continue
				}
				if split[0] == orbName {
					return true
				}
			}
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
