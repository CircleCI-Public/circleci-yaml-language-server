package validate

import (
	"context"
	"fmt"
	"slices"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"

	ast2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/paramref"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/pipelinevalues"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/template"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/yamltree"
)

var (
	stringScalarsQuery = yamltree.MustCompileQuery("(string_scalar) @string")
	blockScalarsQuery  = yamltree.MustCompileQuery("(block_scalar) @string")
	quotedScalarsQuery = yamltree.MustCompileQuery("[(double_quote_scalar) (single_quote_scalar)] @string")
)

func (val Validate) ValidatePipelineParameters() {
	if len(val.Doc.PipelineParameters) == 0 && !position.IsDefaultRange(val.Doc.PipelineParametersRange) {
		val.addDiagnostic(
			diagnostic.EmptySectionWarning(val.Doc.PipelineParametersRange, "parameters"),
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

func (val Validate) checkParamSimpleType(ctx context.Context, param ast2.ParameterValue, stepName string, definedParam ast2.Parameter) {
	if value, ok := param.Value.(string); ok && param.Type == "string" {
		if name, ok := paramref.OnlyPipelineValue(value); ok {
			val.checkPipelineValueType(param, name, stepName, definedParam)
			return
		}
	}

	switch definedParam.GetType() {
	case "string", "boolean", "integer":
		checkParamType(definedParam.GetType(), val, param, stepName, definedParam)
	case "enum":
		if param.Type != "string" {
			val.createParameterError(param, stepName, "string")
			return
		}

		value := param.Value.(string)
		if paramref.IsOnlyParameter(value) {
			checkParamType("enum", val, param, stepName, definedParam)
			return
		}
		if !slices.Contains(definedParam.(ast2.EnumParameter).Enum, value) {
			val.addDiagnostic(diagnostic.Error(
				param.Range,
				fmt.Sprintf("Parameter %s is not a valid value for %s", value, definedParam.GetName()),
			))
		}

	case "executor":
		val.checkExecutorParamValue(ctx, param)

	case "steps":
		values, ok := param.Value.([]ast2.ParameterValue)
		if !ok {
			val.createParameterError(param, stepName, definedParam.GetType())
			return
		}
		for _, value := range values {
			if value.Type == "string" {
				commandName := value.Value.(string)
				_, _, isFunction := val.Doc.FunctionForStep(commandName)

				if !val.isKnownStep(ctx, commandName) && !isFunction {
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
		val.warnNullBodyArguments(values)

	case "env_var_name":
		if param.Type != "string" && param.Type != "integer" {
			val.createParameterError(param, stepName, definedParam.GetType())
			return
		}
		// TODO: check if POSIX_REGEX is valid
	}
}

// checkPipelineValueType checks a parameter given a pipeline value, such as
// << pipeline.number >>, against the type that value will have. Any value can
// be written into a string, and the other types are checked only once the
// config is compiled, so only booleans and integers are checked. A pipeline
// value this does not know is accepted rather than guessed at.
func (val Validate) checkPipelineValueType(param ast2.ParameterValue, name string, stepName string, definedParam ast2.Parameter) {
	wanted := definedParam.GetType()
	if wanted != "boolean" && wanted != "integer" {
		return
	}

	value, known := pipelinevalues.Lookup(name)
	if !known || parameterTypeOf(value) == wanted {
		return
	}

	val.createParameterError(param, stepName, wanted)
}

// parameterTypeOf returns the parameter type a pipeline value fits.
func parameterTypeOf(value pipelinevalues.Value) string {
	switch value.Type {
	case "boolean":
		return "boolean"
	case "uint":
		return "integer"
	default:
		return "string"
	}
}

func checkParamType(paramType string, val Validate, param ast2.ParameterValue, stepName string, definedParam ast2.Parameter) {
	value, isString := param.Value.(string)
	if isString && paramref.IsOnlyParameter(value) {
		paramName, _ := paramref.NameUsedAtPos(val.Doc.Content, param.Range.End)
		pipelineParam, ok := val.Doc.PipelineParameters[paramName]
		if ok && !parameterTypesCompatible(paramType, pipelineParam.GetType()) {
			val.createParameterError(param, stepName, definedParam.GetType())
		}
		return
	}

	// A reference written into a longer string is still a string, and an
	// expression's type is only known once the config is compiled.
	if isString && (paramType == "string" && paramref.ContainsReference(value) || paramref.IsOnlyReference(value)) {
		return
	}

	if param.Type != paramType {
		val.createParameterError(param, stepName, definedParam.GetType())
	}
}

// parameterTypesCompatible reports whether a parameter of type valueType can
// be passed on to one of type definedType. A string, enum or env_var_name
// value can be passed to any of the three: whether the value itself fits is
// only known once the config is compiled.
func parameterTypesCompatible(definedType, valueType string) bool {
	if definedType == valueType {
		return true
	}

	textual := []string{"string", "enum", "env_var_name"}
	return slices.Contains(textual, definedType) && slices.Contains(textual, valueType)
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
	if !parameterTypesCompatible(definedParam.GetType(), paramUsedAsValue.GetType()) {
		val.createParameterError(param, stepName, definedParam.GetType())
	}
}

// CheckIfParamsExist reports the parameters a template uses that aren't
// defined, anywhere in a tag's expression or as a section's name.
func (val Validate) CheckIfParamsExist(ctx context.Context) {
	checkOnNode := func(match *sitter.QueryMatch) {
		for _, capture := range match.Captures {
			node := &capture.Node
			if val.Doc.IsUnderUnreadTopLevelKey(node) {
				continue
			}

			for _, reference := range template.References(val.Doc.GetRawNodeText(node)) {
				var parameters map[string]ast2.Parameter
				var message string
				if name, ok := strings.CutPrefix(reference.Name, "pipeline.parameters."); ok {
					parameters = val.Doc.PipelineParameters
					message = fmt.Sprintf("Pipeline parameter %s is not defined", name)
					if orb, inOrb := val.inlineOrbAt(val.Doc.NodeToRange(node).Start); inOrb {
						parameters = orb.PipelineParameters
						if _, inConfig := val.Doc.PipelineParameters[name]; inConfig {
							message = fmt.Sprintf("Pipeline parameter %s is not defined in orb %s: an inline orb can't use the config's pipeline parameters, so pass it in as a parameter", name, orb.Name)
						}
					}
					reference.Name = name
				} else if name, ok := strings.CutPrefix(reference.Name, "parameters."); ok {
					parameters = val.Doc.GetParamsWithPosition(val.Doc.NodeToRange(node).Start)
					if jobParameters, ok := val.executorArgumentParametersAt(ctx, val.Doc.NodeToRange(node).Start); ok {
						parameters = jobParameters
					}
					message = fmt.Sprintf("Parameter %s is not defined", name)
					reference.Name = name
				} else if name, ok := strings.CutPrefix(reference.Name, "matrix."); ok {
					var inMatrix bool
					parameters, inMatrix = val.matrixParametersAt(val.Doc.NodeToRange(node).Start)
					message = fmt.Sprintf("Matrix parameter %s is not defined", name)
					if !inMatrix {
						message = fmt.Sprintf("matrix.%s is only set in a workflow job with a matrix", name)
					}
					reference.Name = name
				} else {
					continue
				}

				if _, defined := parameters[reference.Name]; defined {
					continue
				}

				val.addDiagnostic(diagnostic.Error(val.rangeInNode(node, reference.Start, reference.End), message))
			}
		}
	}

	stringScalarsQuery.Run(val.Doc.RootNode, checkOnNode)
	blockScalarsQuery.Run(val.Doc.RootNode, checkOnNode)
	quotedScalarsQuery.Run(val.Doc.RootNode, checkOnNode)
}

// inlineOrbAt returns the inline orb declared around pos. An inline orb
// sees only its own pipeline parameters, not the config's.
func (val Validate) inlineOrbAt(pos protocol.Position) (*ast2.OrbInfo, bool) {
	if !position.InRange(val.Doc.OrbsRange, pos) {
		return nil, false
	}

	for _, orb := range val.Doc.Orbs {
		if orb.Url.IsLocal && position.InRange(orb.Range, pos) {
			info, ok := val.Doc.LocalOrbInfo[orb.Name]
			return info, ok
		}
	}

	return nil, false
}

func (val Validate) validateParametersValue(ctx context.Context, paramsValue map[string]ast2.ParameterValue, calledEntity string, entityRange protocol.Range, calledEntityDefinedParams map[string]ast2.Parameter, usableParams map[string]ast2.Parameter) {
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
			val.checkParamSimpleType(ctx, param, calledEntity, calledEntityDefinedParam)
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

func (val Validate) checkExecutorParamValue(ctx context.Context, param ast2.ParameterValue) {
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

	val.validateExecutorReference(ctx, executorName, executorNameRange)
}

// matrixParametersAt returns the matrix parameters set at pos, which are set
// only within a job invocation that has a matrix. It reports whether pos is in
// such an invocation.
func (val Validate) matrixParametersAt(pos protocol.Position) (map[string]ast2.Parameter, bool) {
	for invocation := range invocations(&val.Doc) {
		rng := invocation.JobInvocationRange
		if !invocation.HasMatrix || position.Compare(pos, rng.Start) < 0 || position.Compare(pos, rng.End) > 0 {
			continue
		}
		parameters := map[string]ast2.Parameter{}
		for name := range invocation.MatrixParams {
			parameters[name] = nil
		}
		return parameters, true
	}
	return nil, false
}
