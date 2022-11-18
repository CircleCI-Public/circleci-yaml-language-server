package parser

import (
	"fmt"
	"strconv"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

func (doc *YamlDocument) parseParameters(paramsNode *sitter.Node) map[string]ast.Parameter {
	// paramsNode is of type block_node
	blockMappingNode := GetChildMapping(paramsNode)
	res := make(map[string]ast.Parameter)
	if blockMappingNode == nil {
		return nil
	}

	iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		doc.parseSingleParameter(child, res)
	})

	return res
}

func (doc *YamlDocument) parseSingleParameter(paramNode *sitter.Node, params map[string]ast.Parameter) {
	// paramNode is a block_mapping_pair
	key, value := paramNode.ChildByFieldName("key"), paramNode.ChildByFieldName("value")

	blockMappingNode := GetChildMapping(value)
	if blockMappingNode == nil {
		return
	}

	paramType, paramTypeRange := doc.GetParameterType(value)
	paramName := doc.GetNodeText(key)

	switch paramType {
	case "string":
		param := doc.parseStringParameter(paramName, blockMappingNode)
		param.Range = NodeToRange(paramNode)
		param.NameRange = NodeToRange(key)
		param.TypeRange = paramTypeRange
		params[param.Name] = param
	case "boolean":
		param := doc.parseBooleanParameter(paramName, blockMappingNode)
		param.Range = NodeToRange(paramNode)
		param.NameRange = NodeToRange(key)
		param.TypeRange = paramTypeRange
		params[param.Name] = param
	case "integer":
		param := doc.parseIntegerParameter(paramName, blockMappingNode)
		param.Range = NodeToRange(paramNode)
		param.NameRange = NodeToRange(key)
		param.TypeRange = paramTypeRange
		params[param.Name] = param
	case "enum":
		param := doc.parseEnumParameter(paramName, blockMappingNode)
		param.Range = NodeToRange(paramNode)
		param.NameRange = NodeToRange(key)
		param.TypeRange = paramTypeRange
		params[param.Name] = param
	case "executor":
		param := doc.parseExecutorParameter(paramName, blockMappingNode)
		param.Range = NodeToRange(paramNode)
		param.NameRange = NodeToRange(key)
		param.TypeRange = paramTypeRange
		params[param.Name] = param
	case "steps":
		param := doc.parseStepsParameter(paramName, blockMappingNode)
		param.Range = NodeToRange(paramNode)
		param.NameRange = NodeToRange(key)
		param.TypeRange = paramTypeRange
		params[param.Name] = param
	case "env_var_name":
		param := doc.parseEnvVariableParameter(paramName, blockMappingNode)
		param.Range = NodeToRange(paramNode)
		param.NameRange = NodeToRange(key)
		param.TypeRange = paramTypeRange
		params[param.Name] = param
	default:
		params[paramName] = ast.StringParameter{BaseParameter: ast.BaseParameter{TypeRange: paramTypeRange, Name: paramName, Range: NodeToRange(blockMappingNode)}}
	}
}

func (doc *YamlDocument) GetParameterType(paramNode *sitter.Node) (paramType string, paramTypeRange protocol.Range) {
	// paramNode is a block_node
	blockMappingNode := GetChildMapping(paramNode)
	if blockMappingNode == nil {
		return "", protocol.Range{}
	}

	iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyName := doc.GetNodeText(child.ChildByFieldName("key"))
		valueNode := child.ChildByFieldName("value")
		switch keyName {
		case "type":
			typeRange := NodeToRange(child)
			if valueNode == nil {
				paramTypeRange = protocol.Range{
					Start: typeRange.End,
					End: protocol.Position{
						Line:      typeRange.End.Line,
						Character: typeRange.End.Character + 999,
					},
				}
			} else {
				paramTypeRange = typeRange
			}
			paramType = doc.GetNodeText(valueNode)
		}
	})

	return paramType, paramTypeRange
}

func (doc *YamlDocument) parseStringParameter(paramName string, paramNode *sitter.Node) (stringParam ast.StringParameter) {
	// paramNode is a block_mapping_pair
	stringParam.Name = paramName

	iterateOnBlockMapping(paramNode, func(child *sitter.Node) {
		keyName := doc.GetNodeText(child.ChildByFieldName("key"))
		valueNode := child.ChildByFieldName("value")
		switch keyName {
		case "default":
			stringParam.DefaultRange = getDefaultRange(child)
			stringParam.Default = doc.GetNodeText(valueNode)
			stringParam.HasDefault = true
		case "description":
			stringParam.Description = doc.GetNodeText(valueNode)
		}
	})

	return stringParam
}

func (doc *YamlDocument) parseBooleanParameter(paramName string, paramNode *sitter.Node) (boolParam ast.BooleanParameter) {
	// paramNode is a block_mapping_pair
	boolParam.Name = paramName

	iterateOnBlockMapping(paramNode, func(child *sitter.Node) {
		keyName := doc.GetNodeText(child.ChildByFieldName("key"))
		valueNode := child.ChildByFieldName("value")
		switch keyName {
		case "default":
			boolParam.DefaultRange = getDefaultRange(child)
			boolParam.Default = utils.GetYAMLBooleanValue(doc.GetNodeText(valueNode))
			boolParam.HasDefault = true
		case "description":
			boolParam.Description = doc.GetNodeText(valueNode)
		}
	})

	return boolParam
}

func (doc *YamlDocument) parseIntegerParameter(paramName string, paramNode *sitter.Node) (intParam ast.IntegerParameter) {
	// paramNode is a block_mapping_pair
	intParam.Name = paramName

	iterateOnBlockMapping(paramNode, func(child *sitter.Node) {
		keyName := doc.GetNodeText(child.ChildByFieldName("key"))
		valueNode := child.ChildByFieldName("value")
		switch keyName {
		case "default":
			int, err := strconv.Atoi(doc.GetNodeText(valueNode))
			if err != nil {
				return // TODO: error
			}
			intParam.DefaultRange = getDefaultRange(child)
			intParam.Default = int
			intParam.HasDefault = true
		case "description":
			intParam.Description = doc.GetNodeText(valueNode)
		}
	})

	return intParam
}

func (doc *YamlDocument) parseEnumParameter(paramName string, paramNode *sitter.Node) (enumParam ast.EnumParameter) {
	// paramNode is a block_mapping_pair
	enumParam.Name = paramName

	iterateOnBlockMapping(paramNode, func(child *sitter.Node) {
		keyName := doc.GetNodeText(child.ChildByFieldName("key"))
		valueNode := child.ChildByFieldName("value")
		switch keyName {
		case "default":
			enumParam.DefaultRange = getDefaultRange(child)
			enumParam.Default = doc.GetNodeText(valueNode)
			enumParam.HasDefault = true
		case "description":
			enumParam.Description = doc.GetNodeText(valueNode)
		case "enum":
			enumParam.Enum = doc.getNodeTextArray(valueNode)
		}
	})

	if enumParam.HasDefault && utils.FindInArray(enumParam.Enum, enumParam.Default) == -1 {
		doc.addDiagnostic(utils.CreateErrorDiagnosticFromRange(enumParam.DefaultRange, "Default value is not in enum"))
	}

	return enumParam
}

func (doc *YamlDocument) parseExecutorParameter(paramName string, paramNode *sitter.Node) (executorParam ast.ExecutorParameter) {
	// paramNode is a block_mapping_pair
	executorParam.Name = paramName

	iterateOnBlockMapping(paramNode, func(child *sitter.Node) {
		keyName := doc.GetNodeText(child.ChildByFieldName("key"))
		valueNode := child.ChildByFieldName("value")
		switch keyName {
		case "default":
			executorParam.DefaultRange = getDefaultRange(child)
			executorParam.Default = doc.GetNodeText(valueNode)
			executorParam.HasDefault = true
		case "description":
			executorParam.Description = doc.GetNodeText(valueNode)
		}
	})

	return executorParam
}

func (doc *YamlDocument) parseStepsParameter(paramName string, paramNode *sitter.Node) (stepsParam ast.StepsParameter) {
	// paramNode is a block_mapping_pair
	stepsParam.Name = paramName

	iterateOnBlockMapping(paramNode, func(child *sitter.Node) {
		keyName := doc.GetNodeText(child.ChildByFieldName("key"))
		valueNode := child.ChildByFieldName("value")
		switch keyName {
		case "default":
			stepsNode := GetChildOfType(valueNode, "block_sequence")
			if stepsNode == nil {
				return
			}
			rng := NodeToRange(child)
			astDefault, _ := doc.parseArrayParameterValue(paramName, stepsNode, rng)
			stepsParam.Default = astDefault
			stepsParam.DefaultRange = getDefaultRange(child)
			stepsParam.HasDefault = true
			for _, step := range stepsParam.Default.Value.([]ast.ParameterValue) {
				if step.Type != "steps" {
					doc.addDiagnostic(utils.CreateErrorDiagnosticFromRange(step.Range, "Not a valid step"))
				}
			}
		case "description":
			stepsParam.Description = doc.GetNodeText(valueNode)
		}
	})

	return stepsParam
}

func (doc *YamlDocument) parseEnvVariableParameter(paramName string, paramNode *sitter.Node) (envVariable ast.EnvVariableParameter) {
	// paramNode is a block_mapping_pair
	envVariable.Name = paramName

	iterateOnBlockMapping(paramNode, func(child *sitter.Node) {
		keyName := doc.GetNodeText(child.ChildByFieldName("key"))
		valueNode := child.ChildByFieldName("value")
		switch keyName {
		case "default":
			envVariable.DefaultRange = getDefaultRange(child)
			envVariable.Default = doc.GetNodeText(valueNode)
			envVariable.HasDefault = true
		case "description":
			envVariable.Description = doc.GetNodeText(valueNode)
		}
	})

	return envVariable
}

func (doc *YamlDocument) parseParameterValue(child *sitter.Node) (ast.ParameterValue, error) {
	// flowNode can be either a flow_node or a block_node
	//
	// flowNode is a flow_node, which:
	// - has a single child (plain_scalar), which have a single child too
	//    (string_scalar or boolean_scalar or integer_scalar)
	// - has a single child (flow_sequence or a block sequence) and is an enum
	//
	// flowNode can be a block_node too, and in this case:
	// - has a single child (block_scalar) which is a string but escaped at the
	//   beginning of the string with "|"
	keyNode, valueNode := getKeyValueNodes(child)
	paramName := doc.GetNodeText(keyNode)
	flowNodeChild := valueNode.Child(0)
	if flowNodeChild == nil {
		return ast.ParameterValue{}, fmt.Errorf("error while parsing parameter value")
	}
	rng := NodeToRange(child)
	switch flowNodeChild.Type() {
	case "plain_scalar":
		return doc.parseSimpleParameterValue(paramName, flowNodeChild, rng)
	case "block_scalar":
		return doc.parseSimpleParameterValue(paramName, flowNodeChild, rng)

	case "block_sequence":
		return doc.parseArrayParameterValue(paramName, flowNodeChild, rng)
	case "flow_sequence":
		return doc.parseArrayParameterValue(paramName, flowNodeChild, rng)

	case "double_quote_scalar":
		return ast.ParameterValue{
			Value:      doc.GetNodeText(flowNodeChild),
			ValueRange: NodeToRange(flowNodeChild),
			Name:       paramName,
			Type:       "string",
			Range:      rng,
		}, nil

	case "single_quote_scalar":
		return ast.ParameterValue{
			Value:      doc.GetNodeText(flowNodeChild),
			ValueRange: NodeToRange(flowNodeChild),
			Name:       paramName,
			Type:       "string",
			Range:      rng,
		}, nil

	case "alias":
		return ast.ParameterValue{
			Value:      valueNode.ChildByFieldName("value"),
			Name:       paramName,
			ValueRange: NodeToRange(flowNodeChild),
			Type:       "alias",
			Range:      rng,
		}, nil

	case "block_mapping":
		value := make(map[string]ast.ParameterValue, 0)

		iterateOnBlockMapping(flowNodeChild, func(child *sitter.Node) {
			keyNode, valueNode := getKeyValueNodes(child)

			if keyNode == nil || valueNode == nil {
				return
			}

			key := doc.GetNodeText(keyNode)
			paramValue, err := doc.parseParameterValue(child)

			if err != nil {
				return
			}

			value[key] = paramValue
		})

		return ast.ParameterValue{
			Value:      value,
			Name:       paramName,
			ValueRange: NodeToRange(flowNodeChild),
			Type:       "map",
			Range:      rng,
		}, nil
	}

	return ast.ParameterValue{Name: paramName}, nil // not supported atm by the parser
}

func (doc *YamlDocument) parseArrayParameterValue(paramName string, arrayParamNode *sitter.Node, rng protocol.Range) (ast.ParameterValue, error) {
	// arrayParamNode is a flow_sequence or a block sequence
	values := make([]ast.ParameterValue, 0)
	iterateOnBlockSequence(arrayParamNode, func(child *sitter.Node) {
		if child.Type() == "block_sequence_item" || child.Type() == "flow_node" {
			if isStep(doc, child) {
				steps := doc.parseSingleStep(child)
				values = append(values, ast.ParameterValue{
					Value:      steps,
					ValueRange: NodeToRange(child),
					Name:       paramName,
					Type:       "steps",
				})
			} else {
				param, err := parseEnumParamValue(child, doc, paramName, rng)
				if err != nil {
					// TODO: error
					return
				}
				values = append(values, param)
			}
		}
	})

	return ast.ParameterValue{
		Name:       paramName,
		Value:      values,
		ValueRange: NodeToRange(arrayParamNode),
		Type:       "enum",
		Range:      rng,
	}, nil
}

func parseEnumParamValue(child *sitter.Node, doc *YamlDocument, paramName string, rng protocol.Range) (ast.ParameterValue, error) {
	if child.Type() == "block_sequence_item" {
		child = GetChildOfType(child, "flow_node")
	}
	if child != nil && child.Type() == "flow_node" {
		param, err := doc.parseSimpleParameterValue(paramName, child, rng)
		if err != nil {
			return ast.ParameterValue{}, err
		}
		param.Range = NodeToRange(child)
		return param, nil
	}
	return ast.ParameterValue{}, fmt.Errorf("error while parsing enum parameter value")
}

func isStep(doc *YamlDocument, child *sitter.Node) bool {
	blockNode := GetChildOfType(child, "block_node")
	blockMappingNode := GetChildMapping(blockNode)
	blockMappingPairNode := GetChildOfType(blockMappingNode, "block_mapping_pair")
	return blockMappingPairNode != nil
}

func (doc *YamlDocument) parseSimpleParameterValue(paramName string, simpleParamNode *sitter.Node, rng protocol.Range) (ast.ParameterValue, error) {
	// simpleParamNode's child is either a string_scalar, a boolean_scalar or an integer_scalar
	simpleParamNodeChild := simpleParamNode.Child(0)

	if simpleParamNodeChild == nil {
		return ast.ParameterValue{}, fmt.Errorf("error while parsing simple parameter value")
	}

	switch simpleParamNodeChild.Type() {
	case "double_quote_scalar":
		return ast.ParameterValue{
			Value:      doc.GetNodeText(simpleParamNode),
			ValueRange: NodeToRange(simpleParamNode),
			Name:       paramName,
			Type:       "string",
			Range:      rng,
		}, nil

	case "string_scalar":
		return ast.ParameterValue{
			Value:      doc.GetNodeText(simpleParamNode),
			ValueRange: NodeToRange(simpleParamNode),
			Name:       paramName,
			Type:       "string",
			Range:      rng,
		}, nil

	case "boolean_scalar":
		return ast.ParameterValue{
			Value:      utils.GetYAMLBooleanValue(doc.GetNodeText(simpleParamNode)),
			ValueRange: NodeToRange(simpleParamNode),
			Name:       paramName,
			Type:       "boolean",
			Range:      rng,
		}, nil

	case "integer_scalar":
		rawValue := doc.GetNodeText(simpleParamNode)
		intValue, err := strconv.Atoi(rawValue)

		if err != nil {
			return ast.ParameterValue{}, fmt.Errorf("invalid integer value: %s", rawValue)
		}

		return ast.ParameterValue{
			Value:      intValue,
			ValueRange: NodeToRange(simpleParamNode),
			Name:       paramName,
			Type:       "integer",
			Range:      rng,
		}, nil

	case "plain_scalar":
		return doc.parseSimpleParameterValue(paramName, simpleParamNodeChild, rng)
	}

	return ast.ParameterValue{}, fmt.Errorf("unsupported parameter value type")
}

func getDefaultRange(child *sitter.Node) protocol.Range {
	_, value := getKeyValueNodes(child)

	if value != nil {
		return NodeToRange(child)
	} else {
		defaultRange := NodeToRange(child)
		return protocol.Range{
			Start: defaultRange.Start,
			End: protocol.Position{
				Line:      defaultRange.End.Line,
				Character: defaultRange.End.Character + 999,
			},
		}
	}
}
