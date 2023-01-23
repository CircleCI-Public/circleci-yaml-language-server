package validate

import (
	"fmt"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

func (val Validate) ValidatePipelineParameters() {
	if len(val.Doc.PipelinesParameters) == 0 && !utils.IsDefaultRange(val.Doc.PipelinesParametersRange) {
		val.addDiagnostic(
			utils.CreateEmptyAssignationWarning(val.Doc.PipelinesParametersRange),
		)
	}
}

// Check if the parameter is defined if it's not optional,
// otherwise add a diagnostic error if the needed parameter is not assigned
func (val Validate) checkIfParamAssigned(params map[string]ast.ParameterValue, definedParam ast.Parameter, stepName string, stepRange protocol.Range) bool {
	_, assigned := params[definedParam.GetName()]

	if !assigned && !definedParam.IsOptional() {
		val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
			stepRange,
			fmt.Sprintf("Parameter %s is required for %s", definedParam.GetName(), stepName)))
		return false
	}

	return assigned
}

func (val Validate) checkParamSimpleType(param ast.ParameterValue, stepName string, definedParam ast.Parameter) {
	switch definedParam.GetType() {
	case "string":
		if param.Type != "string" {
			val.createParameterError(param, stepName, definedParam.GetType())
		}

	case "boolean":
		if param.Type != "boolean" {
			val.createParameterError(param, stepName, definedParam.GetType())
		}

	case "integer":
		if param.Type != "integer" {
			val.createParameterError(param, stepName, definedParam.GetType())
		}

	case "enum":
		if param.Type != "string" {
			val.createParameterError(param, stepName, "string")
		}

		value := param.Value.(string)
		if utils.FindInArray(definedParam.(ast.EnumParameter).Enum, value) == -1 {
			val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
				param.Range,
				fmt.Sprintf("Parameter %s is not a valid value for %s", value, definedParam.GetName()),
			))
		}

	case "executor":
		val.checkExecutorParamValue(param)

	case "steps":
		values, ok := param.Value.([]ast.ParameterValue)
		if !ok {
			val.createParameterError(param, stepName, definedParam.GetType())
		}
		for _, value := range values {
			if value.Type == "string" {
				commandName := value.Value.(string)
				_, commandExists := val.Doc.Commands[commandName]

				if !commandExists {
					val.addDiagnostic(
						utils.CreateErrorDiagnosticFromRange(
							value.Range,
							fmt.Sprintf("Cannot find a definition for command named %s", commandName),
						),
					)
				}
			} else if value.Type != "steps" {
				val.createParameterError(value, stepName, definedParam.GetType())
			}
		}

	case "env_variable":
		if param.Type != "string" && param.Type != "integer" {
			val.createParameterError(param, stepName, definedParam.GetType())
		}
		// TODO: check if POSIX_REGEX is valid
	}
}

func (val Validate) checkParamUsedWithParam(param ast.ParameterValue, stepName string, definedParam ast.Parameter, parameters map[string]ast.Parameter) {
	paramName, isPipelineParam := utils.GetParamNameUsedAtPos(val.Doc.Content, param.Range.End)

	var paramUsedAsValue ast.Parameter
	var ok bool
	if isPipelineParam {
		paramUsedAsValue, ok = val.Doc.PipelinesParameters[paramName]
	} else {
		paramUsedAsValue, ok = parameters[paramName]
	}

	if !ok {
		// check already done before in `CheckIfParamsExist`
		return
	}

	if paramUsedAsValue.GetType() != definedParam.GetType() {
		val.createParameterError(param, stepName, definedParam.GetType())
	}
}

func (val Validate) CheckIfParamsExist() {
	checkOnNode := func(match *sitter.QueryMatch) {
		for _, capture := range match.Captures {
			node := capture.Node
			content := val.Doc.GetRawNodeText(node)
			params, err := utils.GetParamsInString(content)

			if err != nil {
				return
			}

			for _, param := range params {
				isPipeline := strings.HasPrefix(param.FullName, "pipeline")

				var parameters map[string]ast.Parameter

				if isPipeline {
					parameters = val.Doc.PipelinesParameters
				} else {
					parameters = val.Doc.GetParamsWithPosition(parser.NodeToRange(node).Start)
				}

				_, parameterFound := parameters[param.Name]

				if parameterFound {
					continue
				}

				diagnosticRange := protocol.Range{
					Start: protocol.Position{
						Line:      param.ParamRange.Start.Line + node.StartPoint().Row,
						Character: param.ParamRange.Start.Character + node.StartPoint().Column,
					},
					End: protocol.Position{
						Line:      param.ParamRange.End.Line + node.StartPoint().Row,
						Character: param.ParamRange.End.Character + node.StartPoint().Column,
					},
				}

				if node.Type() == "block_scalar" {
					// Little difference when the node is a block scalar,
					// We should remove the node Char bonus on the positions

					diagnosticRange.Start.Character -= node.StartPoint().Column
					diagnosticRange.End.Character -= node.StartPoint().Column
				}

				errorMessage := ""

				if isPipeline {
					errorMessage = fmt.Sprintf("Pipeline parameter %s is not defined", param.Name)
				} else {
					errorMessage = fmt.Sprintf("Parameter %s is not defined", param.Name)
				}

				val.addDiagnostic(utils.CreateErrorDiagnosticFromRange(
					diagnosticRange,
					errorMessage,
				))
			}
		}
	}

	parser.ExecQuery(val.Doc.RootNode, "(string_scalar) @string", checkOnNode)
	parser.ExecQuery(val.Doc.RootNode, "(block_scalar) @string", checkOnNode)
}

func (val Validate) validateParametersValue(paramsValue map[string]ast.ParameterValue, calledEntity string, entityRange protocol.Range, definedParams map[string]ast.Parameter, usableParams map[string]ast.Parameter) {
	for _, definedParam := range definedParams {
		// TODO: find a better place to do this
		if definedParam.GetType() == "enum" {
			val.checkEnumTypeDefinition(definedParam.(ast.EnumParameter))
		}

		assigned := val.checkIfParamAssigned(paramsValue, definedParam, calledEntity, entityRange)

		// If the parameter is not assigned but is optional,
		// we don't need to check the parameter
		if !assigned {
			continue
		}

		param := paramsValue[definedParam.GetName()]
		if param.Type == "string" && utils.CheckIfOnlyParamUsed(param.Value.(string)) {
			val.checkParamUsedWithParam(param, calledEntity, definedParam, usableParams)
		} else {
			val.checkParamSimpleType(param, calledEntity, definedParam)
		}
	}

	for _, param := range paramsValue {
		if _, ok := definedParams[param.Name]; !ok {
			val.addDiagnostic(
				utils.CreateErrorDiagnosticFromRange(
					param.Range,
					fmt.Sprintf("Parameter %s is not defined for %s", param.Name, calledEntity),
				),
			)
		}
	}
}

func (val Validate) checkExecutorParamValue(param ast.ParameterValue) {
	executorName := ""
	executorNameRange := param.Range

	if param.Type == "map" {
		nameParam, ok := param.Value.(map[string]ast.ParameterValue)["name"]

		if !ok || nameParam.Type != "string" {
			val.addDiagnostic(
				utils.CreateErrorDiagnosticFromRange(
					param.Range,
					"Missing executor name",
				),
			)
			return
		}

		executorName = nameParam.Value.(string)
		executorNameRange = nameParam.Range
	} else if param.Type == "string" {
		executorName = param.Value.(string)
	}

	if executorName != "" && !val.Doc.DoesExecutorExist(executorName) {
		val.addDiagnostic(
			utils.CreateErrorDiagnosticFromRange(
				executorNameRange,
				fmt.Sprintf(
					"Executor `%s` does not exist",
					executorName,
				),
			),
		)
	}
}
