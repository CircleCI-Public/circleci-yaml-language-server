package validate

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.lsp.dev/protocol"

	ast2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/paramref"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
)

var WHEN_KEYWORDS = []string{
	"on_success",
	"always",
	"on_fail",
}

// AutoRerunDelay validation regex: matches 1-10 minutes or any number of seconds (but not both)
var AUTO_RERUN_DELAY_REGEX = regexp.MustCompile(`^((10|[1-9])m|([1-9][0-9]*)s)$`)

func (val Validate) validateSteps(steps []ast2.Step, name string, jobOrCommandParameters map[string]ast2.Parameter) {
	for _, step := range steps {
		switch step := step.(type) {
		case ast2.Run:
			val.validateRunCommand(step, jobOrCommandParameters)
		case ast2.NamedStep:
			// Function steps are checked by ValidateFunctions.
			if _, _, ok := val.Doc.FunctionForStep(step.Name); ok {
				continue
			}
			val.validateNamedStep(step, jobOrCommandParameters)
		case ast2.Steps:
			val.validateStepSteps(step, name)
		case ast2.Checkout:
			val.validateCheckout(step)
		case ast2.SetupRemoteDocker:
			val.validateSetupRemoteDocker(step)
		}
	}
}

func (val Validate) validateSetupRemoteDocker(step ast2.SetupRemoteDocker) {
	if step.ResourceClass.Text != "" {
		val.addDiagnostic(diagnostic.Warning(
			step.ResourceClass.Range,
			"setup_remote_docker has no resource_class option, so this is ignored.",
		))
	}
}

func (val Validate) validateRunCommand(step ast2.Run, jobOrCommandParameters map[string]ast2.Parameter) {
	if step.IsDeployStep {
		val.addDiagnostic(protocol.Diagnostic{
			Range:    step.Range,
			Message:  protocol.String("The `deploy` step is deprecated. Please use the `run` job instead."),
			Severity: protocol.DiagnosticSeverityWarning,
			Tags:     protocol.NewDiagnosticTags(protocol.DiagnosticTagDeprecated),
		})
	}

	// Validate that background steps cannot use max_auto_reruns or auto_rerun_delay
	if step.Background && (step.MaxAutoReruns != "" || step.AutoRerunDelay != "") {
		val.addDiagnostic(protocol.Diagnostic{
			Range:    step.Range,
			Message:  protocol.String("Background steps cannot use max_auto_reruns or auto_rerun_delay fields"),
			Severity: protocol.DiagnosticSeverityError,
		})
	}

	// Validate that auto_rerun_delay requires max_auto_reruns
	if step.AutoRerunDelay != "" && step.MaxAutoReruns == "" {
		val.addDiagnostic(protocol.Diagnostic{
			Range:    step.Range,
			Message:  protocol.String("auto_rerun_delay requires max_auto_reruns to be specified"),
			Severity: protocol.DiagnosticSeverityError,
		})
	}

	// Validate that max_auto_reruns is between 1 and 5
	if step.MaxAutoReruns != "" && !paramref.ContainsReference(step.MaxAutoReruns) {
		rerunCount, err := strconv.Atoi(step.MaxAutoReruns)
		if err != nil || rerunCount <= 0 || rerunCount > 5 {
			val.addDiagnostic(protocol.Diagnostic{
				Range:    step.Range,
				Message:  protocol.String("max_auto_reruns must be between 1 and 5"),
				Severity: protocol.DiagnosticSeverityError,
			})
		}
	}

	// Validate that auto_rerun_delay conforms to the specific format and duration limits
	if step.AutoRerunDelay != "" && !paramref.ContainsReference(step.AutoRerunDelay) {
		// First check if it's a valid duration
		duration, err := time.ParseDuration(step.AutoRerunDelay)
		if err != nil {
			val.addDiagnostic(protocol.Diagnostic{
				Range:    step.Range,
				Message:  protocol.String("auto_rerun_delay must be a valid duration"),
				Severity: protocol.DiagnosticSeverityError,
			})
		} else {
			// Check if it matches the required format
			if !AUTO_RERUN_DELAY_REGEX.MatchString(step.AutoRerunDelay) {
				val.addDiagnostic(protocol.Diagnostic{
					Range:    step.Range,
					Message:  protocol.String("auto_rerun_delay must be in the format of 1-10 minutes (e.g., '1m', '10m') or any number of seconds (e.g., '30s', '120s')"),
					Severity: protocol.DiagnosticSeverityError,
				})
			}
			// Check if duration exceeds 10 minutes
			if duration > 10*time.Minute {
				val.addDiagnostic(protocol.Diagnostic{
					Range:    step.Range,
					Message:  protocol.String("auto_rerun_delay must not exceed 10 minutes"),
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

	if paramref.IsOnlyParameter(step.When) {
		paramName, isPipelineParam := paramref.NameUsedAtPos(val.Doc.Content, step.WhenRange.End)
		var param ast2.Parameter
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
			case ast2.StringParameter:
				value = param.Default
			default:
				val.addDiagnostic(diagnostic.Error(
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
		val.addDiagnostic(diagnostic.Error(
			step.WhenRange,
			fmt.Sprintf("Invalid when condition: expected `%s`; got `%s`", strings.Join(WHEN_KEYWORDS, "`, `"), value)))
	}
}

func (val Validate) validateNamedStep(step ast2.NamedStep, usableParams map[string]ast2.Parameter) {
	commandExists := val.Doc.DoesJobExist(step.Name) ||
		val.Doc.DoesCommandExist(step.Name) ||
		val.Doc.IsBuiltIn(step.Name) ||
		val.Doc.IsOrbCommand(step.Name, val.Cache) ||
		val.Doc.IsAlias(step.Name)

	if val.Doc.IsFromUnfetchableOrb(step.Name) {
		return
	}

	if !commandExists {
		val.addDiagnostic(diagnostic.Error(
			step.Range,
			fmt.Sprintf("Cannot find declaration for step %s", step.Name)))
	}

	if !val.Doc.IsBuiltIn(step.Name) {
		targetEntityDefinedParams := val.Doc.GetDefinedParams(step.Name, parser.CommandEntity, val.Cache)
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
				Message:  protocol.String("Path must be specified for `store_test_results` step"),
				Range:    step.Range,
				Severity: protocol.DiagnosticSeverityError,
			})
	}
}

func (val Validate) validateStepSteps(step ast2.Steps, name string) {
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
			Message:  protocol.String("Parameter type is not steps"),
			Source:   protocol.NewOptional("cci-language-server"),
		})
	}
}

func (val Validate) validateCheckout(step ast2.Checkout) {
	if step.Method == "" {
		return
	}

	if !slices.Contains(ast2.CheckoutMethods, step.Method) {
		val.addDiagnostic(protocol.Diagnostic{
			Severity: protocol.DiagnosticSeverityError,
			Range:    step.Range,
			Message:  protocol.String(fmt.Sprintf("Checkout method '%s' is invalid", step.Method)),
		})
	}

	if step.Method == "shallow" {
		depth, err := strconv.Atoi(step.Depth)
		if err != nil {
			val.addDiagnostic(protocol.Diagnostic{
				Severity: protocol.DiagnosticSeverityError,
				Range:    step.Range,
				Message:  protocol.String("Checkout depth is not an integer"),
			})
			return
		}
		if depth <= 0 {
			val.addDiagnostic(protocol.Diagnostic{
				Severity: protocol.DiagnosticSeverityError,
				Range:    step.Range,
				Message:  protocol.String("Checkout depth must be a positive integer when using the shallow checkout method"),
			})
		}
	}

	if step.Method != "shallow" && step.Depth != "" {
		val.addDiagnostic(protocol.Diagnostic{
			Severity: protocol.DiagnosticSeverityError,
			Range:    step.Range,
			Message:  protocol.String("Checkout depth can only be used with the shallow checkout method"),
		})
	}

}

func (val Validate) checkIfStepsContainStep(steps []ast2.Step, stepName string) bool {
	return anyStep(steps, func(step ast2.Step) bool {
		return step.GetName() == stepName
	})
}

func (val Validate) checkIfStepsContainOrb(steps []ast2.Step, orbName string) bool {
	return anyStep(steps, val.isStepFromOrb(orbName))
}

func (val Validate) checkIfJobParamContainOrb(params map[string]ast2.ParameterValue, orbName string) bool {
	return anyParamStep(params, val.isStepFromOrb(orbName))
}

func (val Validate) isStepFromOrb(orbName string) func(ast2.Step) bool {
	return func(step ast2.Step) bool {
		name := step.GetName()
		return val.Doc.IsOrbReference(name) && strings.Split(name, "/")[0] == orbName
	}
}

// anyStep reports whether matches holds for any of steps, or for any step
// passed to one of them in a steps parameter, however deeply nested.
func anyStep(steps []ast2.Step, matches func(ast2.Step) bool) bool {
	for _, step := range steps {
		if matches(step) {
			return true
		}

		if named, ok := step.(ast2.NamedStep); ok && anyParamStep(named.Parameters, matches) {
			return true
		}
	}

	return false
}

// anyParamStep is anyStep for the steps passed as parameters, to a step or to
// a job invocation.
//
// Which parameters are steps is only known from the definition, so the
// parser marks a list item as steps only when it has arguments; a bare
// `- name` item comes out as a string. Counting a string as a step name can
// only ever find a use that is not one, which at worst hides an unused
// warning.
func anyParamStep(params map[string]ast2.ParameterValue, matches func(ast2.Step) bool) bool {
	for _, param := range params {
		values, ok := param.Value.([]ast2.ParameterValue)
		if !ok {
			continue
		}

		for _, value := range values {
			switch value.Type {
			case "steps":
				steps, ok := value.Value.([]ast2.Step)
				if ok && anyStep(steps, matches) {
					return true
				}
			case "string":
				name, ok := value.Value.(string)
				if ok && matches(ast2.NamedStep{Name: name}) {
					return true
				}
			}
		}
	}

	return false
}

func (val Validate) checkIfJobUseOrb(job ast2.Job, orbName string) bool {
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
