package validate

import (
	"fmt"
	"slices"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"

	ast2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/paramref"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/yamltree"
)

var (
	stringScalarsQuery = yamltree.MustCompileQuery("(string_scalar) @string")
	blockScalarsQuery  = yamltree.MustCompileQuery("(block_scalar) @string")
)

func (val Validate) ValidatePipelineParameters() {
	if len(val.Doc.PipelineParameters) == 0 && !position.IsDefaultRange(val.Doc.PipelineParametersRange) {
		val.addDiagnostic(
			diagnostic.EmptyAssignationWarning(val.Doc.PipelineParametersRange),
		)
	}
}

// Check if the parameter is defined if it's not optional,
// otherwise add a diagnostic error if the needed parameter is not assigned
func (val Validate) checkIfParamAssigned(params map[string]ast2.ParameterValue, definedParam ast2.Parameter, stepName string, stepRange protocol.Range) bool {
	_, assigned := params[definedParam.GetName()]

	if !assigned && !definedParam.IsOptional() {
		val.addDiagnostic(diagnostic.Error(
			stepRange,
			fmt.Sprintf("Parameter %s is required for %s", definedParam.GetName(), stepName)))
		return false
	}

	return assigned
}

func (val Validate) checkParamSimpleType(param ast2.ParameterValue, stepName string, definedParam ast2.Parameter) {
	switch definedParam.GetType() {
	case "string", "boolean", "integer":
		checkParamType(definedParam.GetType(), val, param, stepName, definedParam)
	case "enum":
		if param.Type != "string" {
			val.createParameterError(param, stepName, "string")
			return
		}

		value := param.Value.(string)
		if !slices.Contains(definedParam.(ast2.EnumParameter).Enum, value) {
			val.addDiagnostic(diagnostic.Error(
				param.Range,
				fmt.Sprintf("Parameter %s is not a valid value for %s", value, definedParam.GetName()),
			))
		}

	case "executor":
		val.checkExecutorParamValue(param)

	case "steps":
		values, ok := param.Value.([]ast2.ParameterValue)
		if !ok {
			val.createParameterError(param, stepName, definedParam.GetType())
			return
		}
		for _, value := range values {
			if value.Type == "string" {
				commandName := value.Value.(string)
				_, commandExists := val.Doc.Commands[commandName]

				if !commandExists {
					val.addDiagnostic(
						diagnostic.Error(
							value.Range,
							fmt.Sprintf("Cannot find a definition for command named %s", commandName),
						),
					)
				}
			} else if value.Type != "steps" {
				val.createParameterError(value, stepName, definedParam.GetType())
			}
		}

	case "env_var_name":
		if param.Type != "string" && param.Type != "integer" {
			val.createParameterError(param, stepName, definedParam.GetType())
			return
		}
		// TODO: check if POSIX_REGEX is valid
	}
}

func checkParamType(paramType string, val Validate, param ast2.ParameterValue, stepName string, definedParam ast2.Parameter) {
	paramName, _ := paramref.NameUsedAtPos(val.Doc.Content, param.Range.End)
	if paramName != "" {
		pipelineParam, ok := val.Doc.PipelineParameters[paramName]
		if ok && pipelineParam.GetType() != paramType {
			val.createParameterError(param, stepName, definedParam.GetType())
		}
	} else if param.Type != paramType {
		val.createParameterError(param, stepName, definedParam.GetType())
	}
}

func (val Validate) checkParamUsedWithParam(param ast2.ParameterValue, stepName string, definedParam ast2.Parameter, parameters map[string]ast2.Parameter) {
	paramName, isPipelineParam := paramref.NameUsedAtPos(val.Doc.Content, param.Range.End)

	var paramUsedAsValue ast2.Parameter
	var ok bool
	if isPipelineParam {
		paramUsedAsValue, ok = val.Doc.PipelineParameters[paramName]
	} else {
		paramUsedAsValue, ok = parameters[paramName]
	}

	if !ok {
		// check already done before in `CheckIfParamsExist`
		return
	}
	definedType := definedParam.GetType()
	valueType := paramUsedAsValue.GetType()
	if definedType != valueType && (definedType != "string" || valueType != "enum") { // String params can accept "string" or "enum"
		val.createParameterError(param, stepName, definedParam.GetType())
	}
}

func (val Validate) CheckIfParamsExist() {
	checkOnNode := func(match *sitter.QueryMatch) {
		for _, capture := range match.Captures {
			node := &capture.Node
			content := val.Doc.GetRawNodeText(node)
			params, err := paramref.InString(content)
			if err != nil {
				return
			}

			for _, param := range params {
				isPipeline := strings.HasPrefix(param.FullName, "pipeline")

				var parameters map[string]ast2.Parameter

				if isPipeline {
					parameters = val.Doc.PipelineParameters
				} else {
					parameters = val.Doc.GetParamsWithPosition(val.Doc.NodeToRange(node).Start)
				}

				_, parameterFound := parameters[param.Name]

				if parameterFound {
					continue
				}

				diagnosticRange := protocol.Range{
					Start: protocol.Position{
						Line:      param.ParamRange.Start.Line + position.Start(node).Line,
						Character: param.ParamRange.Start.Character + position.Start(node).Character,
					},
					End: protocol.Position{
						Line:      param.ParamRange.End.Line + position.Start(node).Line,
						Character: param.ParamRange.End.Character + position.Start(node).Character,
					},
				}

				if node.Kind() == "block_scalar" {
					// Little difference when the node is a block scalar,
					// We should remove the node Char bonus on the positions

					diagnosticRange.Start.Character -= position.Start(node).Character
					diagnosticRange.End.Character -= position.Start(node).Character
				}

				errorMessage := ""

				if isPipeline {
					errorMessage = fmt.Sprintf("Pipeline parameter %s is not defined", param.Name)
				} else {
					errorMessage = fmt.Sprintf("Parameter %s is not defined", param.Name)
				}

				val.addDiagnostic(diagnostic.Error(
					diagnosticRange,
					errorMessage,
				))
			}
		}
	}

	stringScalarsQuery.Run(val.Doc.RootNode, checkOnNode)
	blockScalarsQuery.Run(val.Doc.RootNode, checkOnNode)
}

func (val Validate) validateParametersValue(paramsValue map[string]ast2.ParameterValue, calledEntity string, entityRange protocol.Range, calledEntityDefinedParams map[string]ast2.Parameter, usableParams map[string]ast2.Parameter) {
	for _, calledEntityDefinedParam := range calledEntityDefinedParams {
		// TODO: find a better place to do this
		if calledEntityDefinedParam.GetType() == "enum" {
			val.checkEnumTypeDefinition(calledEntityDefinedParam.(ast2.EnumParameter))
		}

		assigned := val.checkIfParamAssigned(paramsValue, calledEntityDefinedParam, calledEntity, entityRange)

		// If the parameter is not assigned but is optional,
		// we don't need to check the parameter
		if !assigned {
			continue
		}

		param := paramsValue[calledEntityDefinedParam.GetName()]
		if param.Type == "string" && paramref.IsOnlyParameter(param.Value.(string)) {
			val.checkParamUsedWithParam(param, calledEntity, calledEntityDefinedParam, usableParams)
		} else {
			val.checkParamSimpleType(param, calledEntity, calledEntityDefinedParam)
		}
	}

	for _, param := range paramsValue {
		if _, ok := calledEntityDefinedParams[param.Name]; !ok {
			val.addDiagnostic(
				diagnostic.Error(
					param.Range,
					fmt.Sprintf("Parameter %s is not defined for %s", param.Name, calledEntity),
				),
			)
		}
	}
}

func (val Validate) checkExecutorParamValue(param ast2.ParameterValue) {
	executorName := ""
	executorNameRange := param.Range

	switch param.Type {
	case "map":
		nameParam, ok := param.Value.(map[string]ast2.ParameterValue)["name"]

		if !ok || nameParam.Type != "string" {
			val.addDiagnostic(
				diagnostic.Error(
					param.Range,
					"Missing executor name",
				),
			)
			return
		}

		executorName = nameParam.Value.(string)
		executorNameRange = nameParam.Range
	case "string":
		executorName = param.Value.(string)
	}

	val.validateExecutorReference(executorName, executorNameRange)
}
