package parser

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"

	ast2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/yamlbool"
)

func (doc *YamlDocument) parseParameters(paramsNode *sitter.Node) map[string]ast2.Parameter {
	// paramsNode is of type block_node
	blockMappingNode := GetChildMapping(paramsNode)
	res := make(map[string]ast2.Parameter)
	if blockMappingNode == nil {
		return nil
	}

	doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		doc.parseSingleParameter(child, res)
	})

	return res
}

func (doc *YamlDocument) parseSingleParameter(paramNode *sitter.Node, params map[string]ast2.Parameter) {
	// paramNode is a block_mapping_pair
	keyNode, valueNode := doc.GetKeyValueNodes(paramNode)

	if keyNode == nil {
		return
	}
	if strings.TrimSpace(doc.GetNodeText(keyNode)) == "" {
		doc.addDiagnostic(diagnostic.ErrorFromNode(keyNode, "Parameters must be named"))
		return
	}

	blockMappingNode := GetChildMapping(valueNode)
	if blockMappingNode == nil {
		return
	}

	paramType, paramTypeRange := doc.GetParameterType(valueNode)
	paramName := doc.GetNodeText(keyNode)

	switch paramType {
	case "string":
		param := doc.parseStringParameter(paramName, blockMappingNode)
		param.Range = doc.NodeToRange(paramNode)
		param.NameRange = doc.NodeToRange(keyNode)
		param.TypeRange = paramTypeRange
		params[param.Name] = param
	case "boolean":
		param := doc.parseBooleanParameter(paramName, blockMappingNode)
		param.Range = doc.NodeToRange(paramNode)
		param.NameRange = doc.NodeToRange(keyNode)
		param.TypeRange = paramTypeRange
		params[param.Name] = param
	case "integer":
		param := doc.parseIntegerParameter(paramName, blockMappingNode)
		param.Range = doc.NodeToRange(paramNode)
		param.NameRange = doc.NodeToRange(keyNode)
		param.TypeRange = paramTypeRange
		params[param.Name] = param
	case "enum":
		param := doc.parseEnumParameter(paramName, blockMappingNode)
		param.Range = doc.NodeToRange(paramNode)
		param.NameRange = doc.NodeToRange(keyNode)
		param.TypeRange = paramTypeRange
		params[param.Name] = param
	case "executor":
		param := doc.parseExecutorParameter(paramName, blockMappingNode)
		param.Range = doc.NodeToRange(paramNode)
		param.NameRange = doc.NodeToRange(keyNode)
		param.TypeRange = paramTypeRange
		params[param.Name] = param
	case "steps":
		param := doc.parseStepsParameter(paramName, blockMappingNode)
		param.Range = doc.NodeToRange(paramNode)
		param.NameRange = doc.NodeToRange(keyNode)
		param.TypeRange = paramTypeRange
		params[param.Name] = param
	case "env_var_name":
		param := doc.parseEnvVariableParameter(paramName, blockMappingNode)
		param.Range = doc.NodeToRange(paramNode)
		param.NameRange = doc.NodeToRange(keyNode)
		param.TypeRange = paramTypeRange
		params[param.Name] = param
	default:
		params[paramName] = ast2.StringParameter{
			BaseParameter: ast2.BaseParameter{
				Name:      paramName,
				NameRange: doc.NodeToRange(keyNode),
				Range:     doc.NodeToRange(blockMappingNode),
				TypeRange: paramTypeRange,
			},
		}
	}
}

func (doc *YamlDocument) GetParameterType(paramNode *sitter.Node) (paramType string, paramTypeRange protocol.Range) {
	// paramNode is a block_node
	blockMappingNode := GetChildMapping(paramNode)
	if blockMappingNode == nil {
		return "", protocol.Range{}
	}

	doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
		keyName := doc.GetNodeText(keyNode)
		switch keyName {
		case "type":
			typeRange := doc.NodeToRange(child)
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

func (doc *YamlDocument) parseStringParameter(paramName string, paramNode *sitter.Node) (stringParam ast2.StringParameter) {
	// paramNode is a block_mapping_pair
	stringParam.Name = paramName

	doc.iterateOnBlockMapping(paramNode, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
		keyName := doc.GetNodeText(keyNode)
		switch keyName {
		case "default":
			stringParam.DefaultRange = doc.getDefaultParameterRange(child)
			stringParam.Default = doc.GetNodeText(GetFirstChild(valueNode))
			stringParam.HasDefault = true
		case "description":
			stringParam.Description = doc.parseDescription(valueNode)
		}
	})

	return stringParam
}

func (doc *YamlDocument) parseBooleanParameter(paramName string, paramNode *sitter.Node) (boolParam ast2.BooleanParameter) {
	// paramNode is a block_mapping_pair
	boolParam.Name = paramName

	doc.iterateOnBlockMapping(paramNode, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
		keyName := doc.GetNodeText(keyNode)
		switch keyName {
		case "default":
			boolParam.DefaultRange = doc.getDefaultParameterRange(child)
			boolParam.Default = yamlbool.Value(doc.GetNodeText(valueNode))
			boolParam.HasDefault = true
		case "description":
			boolParam.Description = doc.parseDescription(valueNode)
		}
	})

	return boolParam
}

func (doc *YamlDocument) parseIntegerParameter(paramName string, paramNode *sitter.Node) (intParam ast2.IntegerParameter) {
	// paramNode is a block_mapping_pair
	intParam.Name = paramName

	doc.iterateOnBlockMapping(paramNode, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
		keyName := doc.GetNodeText(keyNode)
		switch keyName {
		case "default":
			int, err := strconv.Atoi(doc.GetNodeText(valueNode))
			if err != nil {
				return // TODO: error
			}
			intParam.DefaultRange = doc.getDefaultParameterRange(child)
			intParam.Default = int
			intParam.HasDefault = true
		case "description":
			intParam.Description = doc.parseDescription(valueNode)
		}
	})

	return intParam
}

func (doc *YamlDocument) parseEnumParameter(paramName string, paramNode *sitter.Node) (enumParam ast2.EnumParameter) {
	// paramNode is a block_mapping_pair
	enumParam.Name = paramName

	doc.iterateOnBlockMapping(paramNode, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
		keyName := doc.GetNodeText(keyNode)
		switch keyName {
		case "default":
			enumParam.DefaultRange = doc.getDefaultParameterRange(child)
			enumParam.Default = doc.GetNodeText(GetFirstChild(valueNode))
			enumParam.HasDefault = true
		case "description":
			enumParam.Description = doc.parseDescription(valueNode)
		case "enum":
			enumParam.Enum = doc.getNodeTextArray(valueNode)
		}
	})

	if enumParam.HasDefault && !slices.Contains(enumParam.Enum, enumParam.Default) {
		doc.addDiagnostic(diagnostic.Error(enumParam.DefaultRange, "Default value is not in enum"))
	}

	return enumParam
}

func (doc *YamlDocument) parseExecutorParameter(paramName string, paramNode *sitter.Node) (executorParam ast2.ExecutorParameter) {
	// paramNode is a block_mapping_pair
	executorParam.Name = paramName

	doc.iterateOnBlockMapping(paramNode, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
		keyName := doc.GetNodeText(keyNode)
		switch keyName {
		case "default":
			executorParam.DefaultRange = doc.getDefaultParameterRange(child)
			executorParam.Default = doc.GetNodeText(GetFirstChild(valueNode))
			executorParam.HasDefault = true
		case "description":
			executorParam.Description = doc.parseDescription(valueNode)
		}
	})

	return executorParam
}

func (doc *YamlDocument) parseStepsParameter(paramName string, paramNode *sitter.Node) (stepsParam ast2.StepsParameter) {
	// paramNode is a block_mapping_pair
	stepsParam.Name = paramName

	doc.iterateOnBlockMapping(paramNode, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
		keyName := doc.GetNodeText(keyNode)
		switch keyName {
		case "default":
			stepsNode := GetChildSequence(valueNode)
			if stepsNode == nil {
				return
			}
			rng := doc.NodeToRange(child)
			astDefault, _ := doc.parseArrayParameterValue(paramName, stepsNode, rng, true)
			stepsParam.Default = astDefault
			stepsParam.DefaultRange = doc.getDefaultParameterRange(child)
			stepsParam.HasDefault = true
			for _, step := range stepsParam.Default.Value.([]ast2.ParameterValue) {
				if step.Type != "steps" {
					doc.addDiagnostic(diagnostic.Error(step.Range, "Not a valid step"))
				}
			}
		case "description":
			stepsParam.Description = doc.parseDescription(valueNode)
		}
	})

	return stepsParam
}

func (doc *YamlDocument) parseEnvVariableParameter(paramName string, paramNode *sitter.Node) (envVariable ast2.EnvVariableParameter) {
	// paramNode is a block_mapping_pair
	envVariable.Name = paramName

	doc.iterateOnBlockMapping(paramNode, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
		keyName := doc.GetNodeText(keyNode)
		switch keyName {
		case "default":
			envVariable.DefaultRange = doc.getDefaultParameterRange(child)
			envVariable.Default = doc.GetNodeText(GetFirstChild(valueNode))
			envVariable.HasDefault = true
		case "description":
			envVariable.Description = doc.parseDescription(valueNode)
		}
	})

	return envVariable
}

func (doc *YamlDocument) parseParameterValue(child *sitter.Node) (ast2.ParameterValue, error) {
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
	keyNode, valueNode := doc.GetKeyValueNodes(child)
	paramName := doc.GetNodeText(keyNode)

	if keyNode == nil {
		return ast2.ParameterValue{}, fmt.Errorf("key not defined")
	}

	if valueNode == nil {
		diag := diagnostic.WarningFromNode(
			keyNode,
			"No value defined for the parameter",
		)
		doc.addDiagnostic(diag)
		return ast2.ParameterValue{}, fmt.Errorf("no parameter value")
	}

	flowNodeChild := GetFirstChild(valueNode)
	if flowNodeChild == nil {
		return ast2.ParameterValue{}, fmt.Errorf("error while parsing parameter value")
	}
	rng := doc.NodeToRange(child)
	switch flowNodeChild.Kind() {
	case "plain_scalar":
		return doc.parseSimpleParameterValue(paramName, flowNodeChild, rng)
	case "block_scalar":
		return doc.parseSimpleParameterValue(paramName, flowNodeChild, rng)

	case "block_sequence":
		return doc.parseArrayParameterValue(paramName, flowNodeChild, rng, false)
	case "flow_sequence":
		return doc.parseArrayParameterValue(paramName, flowNodeChild, rng, false)

	case "double_quote_scalar":
		return ast2.ParameterValue{
			Value:      doc.GetNodeText(flowNodeChild),
			ValueRange: doc.NodeToRange(flowNodeChild),
			Name:       paramName,
			Type:       "string",
			Range:      rng,
		}, nil

	case "single_quote_scalar":
		return ast2.ParameterValue{
			Value:      doc.GetNodeText(flowNodeChild),
			ValueRange: doc.NodeToRange(flowNodeChild),
			Name:       paramName,
			Type:       "string",
			Range:      rng,
		}, nil

	case "alias":
		return ast2.ParameterValue{
			Value:      valueNode.ChildByFieldName("value"),
			Name:       paramName,
			ValueRange: doc.NodeToRange(flowNodeChild),
			Type:       "alias",
			Range:      rng,
		}, nil

	case "block_mapping":
		value := make(map[string]ast2.ParameterValue, 0)

		doc.iterateOnBlockMapping(flowNodeChild, func(child *sitter.Node) {
			keyNode, valueNode := doc.GetKeyValueNodes(child)

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

		return ast2.ParameterValue{
			Value:      value,
			Name:       paramName,
			ValueRange: doc.NodeToRange(flowNodeChild),
			Type:       "map",
			Range:      rng,
		}, nil
	}

	return ast2.ParameterValue{Name: paramName}, nil // not supported atm by the parser
}

func (doc *YamlDocument) parseArrayParameterValue(paramName string, arrayParamNode *sitter.Node, rng protocol.Range, forceSteps bool) (ast2.ParameterValue, error) {
	// arrayParamNode is a flow_sequence or a block sequence
	values := make([]ast2.ParameterValue, 0)
	iterateOnBlockSequence(arrayParamNode, func(child *sitter.Node) {
		if child.Kind() == "block_sequence_item" || child.Kind() == "flow_node" {
			if isStep(doc, child) || forceSteps {
				steps := doc.parseSingleStep(child)
				values = append(values, ast2.ParameterValue{
					Value:      steps,
					ValueRange: doc.NodeToRange(child),
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

	return ast2.ParameterValue{
		Name:       paramName,
		Value:      values,
		ValueRange: doc.NodeToRange(arrayParamNode),
		Type:       "enum",
		Range:      rng,
	}, nil
}

func parseEnumParamValue(child *sitter.Node, doc *YamlDocument, paramName string, rng protocol.Range) (ast2.ParameterValue, error) {
	if child.Kind() == "block_sequence_item" {
		child = GetChildOfType(child, "flow_node")
	}
	if child != nil && child.Kind() == "flow_node" {
		param, err := doc.parseSimpleParameterValue(paramName, child, rng)
		if err != nil {
			return ast2.ParameterValue{}, err
		}
		param.Range = doc.NodeToRange(child)
		return param, nil
	}
	return ast2.ParameterValue{}, fmt.Errorf("error while parsing enum parameter value")
}

func isStep(doc *YamlDocument, child *sitter.Node) bool {
	blockNode := GetChildOfType(child, "block_node")
	blockMappingNode := GetChildMapping(blockNode)
	blockMappingPairNode := GetChildOfType(blockMappingNode, "block_mapping_pair")
	return blockMappingPairNode != nil
}

func (doc *YamlDocument) parseSimpleParameterValue(paramName string, simpleParamNode *sitter.Node, rng protocol.Range) (ast2.ParameterValue, error) {
	// simpleParamNode's child is either a string_scalar, a boolean_scalar or an integer_scalar
	simpleParamNodeChild := GetFirstChild(simpleParamNode)

	if simpleParamNodeChild == nil {
		return ast2.ParameterValue{}, fmt.Errorf("error while parsing simple parameter value")
	}

	// This is needed if a parameter is written such as :
	//     param: >
	//       value
	if simpleParamNodeChild.Kind() == ">" || simpleParamNodeChild.Kind() == "|" {
		simpleParamNodeChild = simpleParamNode
	}

	switch simpleParamNodeChild.Kind() {
	case "double_quote_scalar":
		return ast2.ParameterValue{
			Value:      doc.GetNodeText(simpleParamNode),
			ValueRange: doc.NodeToRange(simpleParamNode),
			Name:       paramName,
			Type:       "string",
			Range:      rng,
		}, nil

	case "string_scalar":
		return ast2.ParameterValue{
			Value:      doc.GetNodeText(simpleParamNode),
			ValueRange: doc.NodeToRange(simpleParamNode),
			Name:       paramName,
			Type:       "string",
			Range:      rng,
		}, nil

	case "block_scalar":
		return ast2.ParameterValue{
			Value:      doc.GetNodeText(simpleParamNode),
			ValueRange: doc.NodeToRange(simpleParamNode),
			Name:       paramName,
			Type:       "string",
			Range:      rng,
		}, nil

	case "boolean_scalar":
		return ast2.ParameterValue{
			Value:      yamlbool.Value(doc.GetNodeText(simpleParamNode)),
			ValueRange: doc.NodeToRange(simpleParamNode),
			Name:       paramName,
			Type:       "boolean",
			Range:      rng,
		}, nil

	case "integer_scalar":
		rawValue := doc.GetNodeText(simpleParamNode)
		intValue, err := strconv.Atoi(rawValue)
		if err != nil {
			return ast2.ParameterValue{}, fmt.Errorf("invalid integer value: %s", rawValue)
		}

		return ast2.ParameterValue{
			Value:      intValue,
			ValueRange: doc.NodeToRange(simpleParamNode),
			Name:       paramName,
			Type:       "integer",
			Range:      rng,
		}, nil

	case "plain_scalar":
		return doc.parseSimpleParameterValue(paramName, simpleParamNodeChild, rng)
	}

	return ast2.ParameterValue{}, fmt.Errorf("unsupported parameter value type")
}

func (doc *YamlDocument) getDefaultParameterRange(child *sitter.Node) protocol.Range {
	_, value := doc.GetKeyValueNodes(child)

	if value != nil {
		return doc.NodeToRange(child)
	}

	defaultRange := doc.NodeToRange(child)
	return protocol.Range{
		Start: defaultRange.Start,
		End: protocol.Position{
			Line:      defaultRange.End.Line,
			Character: defaultRange.End.Character + 999,
		},
	}
}
