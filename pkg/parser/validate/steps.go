package validate

import (
	"fmt"
	"strings"

	"github.com/circleci/circleci-yaml-language-server/pkg/ast"
	"github.com/circleci/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func (val Validate) validateSteps(steps []ast.Step, name string) error {
	for _, step := range steps {
		switch step := step.(type) {
		case ast.NamedStep:
			if !val.Doc.DoesJobExist(step.Name) && !val.Doc.DoesCommandExist(step.Name) &&
				!val.Doc.IsBuiltIn(step.Name) && !val.Doc.IsOrb(step.Name) && !val.Doc.IsAlias(step.Name) {
				val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
					step.Range,
					fmt.Sprintf("Cannot find declaration for step %s", step.Name)))
			}

			if !val.Doc.IsOrb(step.Name) && !val.Doc.IsBuiltIn(step.Name) {
				definedParams := val.Doc.GetDefinedParams(step.Name)
				val.validateParametersValue(
					step.Parameters,
					step.Name,
					step.Range,
					definedParams,
					definedParams,
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
		case ast.Steps:
			if !val.Doc.DoesCommandExist(name) {
				return nil
			}
			command := val.Doc.Commands[name]
			parameter, ok := command.Parameters[step.Name]
			if !ok {
				return nil
			}
			parameterType := parameter.GetType()
			fmt.Println("parameterType", parameterType)
			if parameterType != "steps" {
				val.addDiagnostic(protocol.Diagnostic{
					Severity: protocol.DiagnosticSeverityError,
					Range:    step.Range,
					Message:  "Parameter type is not steps",
					Source:   "cci-language-server",
				})
			}
		}
	}
	return nil
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
		if val.Doc.IsOrb(step.GetName()) && strings.Split(step.GetName(), "/")[0] == orbName {
			return true
		}
	}

	return false
}
