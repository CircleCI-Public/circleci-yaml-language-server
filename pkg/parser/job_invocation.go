package parser

import (
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

func (doc *YamlDocument) parseJobInvocations(jobsInvocationsNode *sitter.Node) []ast.JobInvocation {
	blockSequenceNode := GetChildSequence(jobsInvocationsNode)
	jobInvocations := []ast.JobInvocation{}
	if blockSequenceNode == nil {
		return jobInvocations
	}

	iterateOnBlockSequence(blockSequenceNode, func(child *sitter.Node) {
		jobInvocations = append(jobInvocations, doc.parseSingleJobInvocation(child))
	})
	return jobInvocations
}

func (doc *YamlDocument) parseSingleJobInvocation(jobInvocationNode *sitter.Node) ast.JobInvocation {
	res := ast.JobInvocation{MatrixParams: make(map[string][]ast.ParameterValue), Parameters: make(map[string]ast.ParameterValue)}
	if jobInvocationNode == nil {
		return res
	}
	res.JobInvocationRange = doc.NodeToRange(jobInvocationNode)
	if jobInvocationNode.Type() != "block_sequence_item" {
		return res
	}

	if jobInvocationNode.ChildCount() == 1 {
		res.JobNameRange = protocol.Range{
			Start: protocol.Position{
				Line:      jobInvocationNode.StartPoint().Row,
				Character: jobInvocationNode.StartPoint().Column + 1,
			},
			End: protocol.Position{
				Line:      jobInvocationNode.StartPoint().Row,
				Character: jobInvocationNode.StartPoint().Column + 2,
			},
		}
		return res
	}

	element := jobInvocationNode.Child(1) // element is either flow_node or block_node
	res.Parameters = make(map[string]ast.ParameterValue)
	res.HasMatrix = false

	if alias := GetChildOfType(element, "alias"); alias != nil {
		aliasName := strings.TrimPrefix(doc.GetNodeText(alias), "*")
		anchor, ok := doc.YamlAnchors[aliasName]

		if !ok {
			return res
		}

		element = anchor.ValueNode
	}

	if element != nil && element.Type() == "flow_node" {
		name := GetChildOfType(element, "plain_scalar")
		res.JobName = doc.GetNodeText(name)
		res.JobNameRange = doc.NodeToRange(element)
		res.StepName = res.JobName
		res.StepNameRange = res.JobNameRange
		return res
	} else { // block_node
		blockMappingNode := GetChildOfType(element, "block_mapping")
		blockMappingPair := GetChildOfType(blockMappingNode, "block_mapping_pair")
		key, value := doc.GetKeyValueNodes(blockMappingPair)
		if key == nil || value == nil {
			return res
		}

		res.JobNameRange = doc.NodeToRange(key)
		res.JobName = doc.GetNodeText(key)
		res.StepNameRange = doc.NodeToRange(key)
		res.StepName = doc.GetNodeText(key)
		blockMappingNode = GetChildOfType(value, "block_mapping")

		doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
			if child != nil {
				keyNode, valueNode := doc.GetKeyValueNodes(child)
				keyName := doc.GetNodeText(keyNode)

				if keyName == "" || valueNode == nil {
					return
				}

				switch keyName {
				case "type":
					res.Type = doc.GetNodeText(valueNode)
					res.TypeRange = doc.NodeToRange(valueNode)
				case "requires":
					res.Requires = doc.parseSingleJobRequires(valueNode)
				case "name":
					res.StepNameRange = doc.NodeToRange(valueNode)
					res.StepName = doc.GetNodeText(GetFirstChild(valueNode))
				case "context":
					res.Context = doc.parseContext(valueNode)
				case "filters":
				case "branches":
				case "tags":
				case "matrix":
					res.HasMatrix = true
					res.MatrixRange = doc.NodeToRange(child)
					matrixParams, alias := doc.parseMatrixAttributes(valueNode)
					res.MatrixParams = matrixParams
					if alias != "" {
						res.StepName = alias
					}
				case "serial-group":
					res.SerialGroup = doc.GetNodeText(valueNode)
					res.SerialGroupRange = doc.NodeToRange(valueNode)
				case "override-with":
					res.OverrideWith = doc.GetNodeText(valueNode)
					res.OverrideWithRange = doc.NodeToRange(valueNode)

				case "pre-steps":
					res.PreStepsRange = doc.NodeToRange(child)
					res.PreSteps = doc.parseSteps(valueNode)

				case "post-steps":
					res.PostStepsRange = doc.NodeToRange(child)
					res.PostSteps = doc.parseSteps(valueNode)

				default:
					paramValue, err := doc.parseParameterValue(child)
					if err != nil {
						return
					}
					res.Parameters[keyName] = paramValue
				}
			}
		})
		return res
	}
}

func (doc *YamlDocument) parseContext(node *sitter.Node) []ast.TextAndRange {
	if node.ChildCount() != 1 {
		return []ast.TextAndRange{}
	}

	if node.Type() == "flow_node" && node.ChildCount() == 1 && node.Child(0).Type() == "plain_scalar" {
		return []ast.TextAndRange{doc.GetNodeTextWithRange(node)}
	}

	return doc.getNodeTextArrayWithRange(node)
}

func (doc *YamlDocument) parseSingleJobRequires(requiresNode *sitter.Node) []ast.Require {
	blockSequenceNode := GetChildSequence(requiresNode)
	res := make([]ast.Require, 0, requiresNode.ChildCount())

	if blockSequenceNode == nil {
		return res
	}

	iterateOnBlockSequence(blockSequenceNode, func(requiresItemNode *sitter.Node) {
		getRequire := func(node *sitter.Node) ast.Require {
			defaultStatus := []string{"success"}
			if alias := GetChildOfType(node, "alias"); alias != nil {
				anchor, ok := doc.YamlAnchors[strings.TrimLeft(doc.GetNodeText(alias), "*")]
				if !ok {
					return ast.Require{Name: ""}
				}
				anchorValueNode := GetFirstChild(anchor.ValueNode)
				text := doc.GetNodeText(anchorValueNode)
				return ast.Require{
					Name:        text,
					Status:      defaultStatus,
					Range:       doc.NodeToRange(anchorValueNode),
					StatusRange: doc.NodeToRange(node),
				}
			} else {
				return ast.Require{
					Name:        doc.GetNodeText(node),
					Status:      defaultStatus,
					Range:       doc.NodeToRange(node),
					StatusRange: doc.NodeToRange(node),
				}
			}
		}

		// If blockSequenceNode is a flow_sequence, then requiresItemNode is directly a flow_node
		if requiresItemNode.Type() == "flow_node" {
			res = append(res, getRequire(requiresItemNode))
		} else {
			// But if blockSequenceNode is a block_sequence, then requiresItemNode is a block_sequence_item
			// The first child of requiresItemNode is the hyphen node, the second child is what we need
			element := requiresItemNode.Child(1)
			// If the second child is a flow_node, then it is a simple require
			if element != nil && element.Type() == "flow_node" {
				res = append(res, getRequire(element))
			} else {
				// Otherwise the second child is a block_mapping, then it is a require with status
				blockMappingNode := GetChildOfType(element, "block_mapping")
				blockMappingPair := GetChildOfType(blockMappingNode, "block_mapping_pair")
				key, value := doc.GetKeyValueNodes(blockMappingPair)

				if key == nil || value == nil {
					return
				}
				if GetFirstChild(value).Type() == "plain_scalar" {
					status := make([]string, 1)
					status[0] = doc.GetNodeText(value)
					res = append(res, ast.Require{
						Name:        doc.GetNodeText(key),
						Status:      status,
						Range:       doc.NodeToRange(key),
						StatusRange: doc.NodeToRange(value),
					})
				} else {
					statusesNode := GetFirstChild(value)
					status := make([]string, 0, statusesNode.ChildCount())
					isBlockSequence := false
					iterateOnBlockSequence(statusesNode, func(statusItemNode *sitter.Node) {
						if statusItemNode.Type() == "flow_node" {
							status = append(status, doc.GetNodeText(statusItemNode))
						}
						if statusItemNode.Type() == "block_sequence_item" {
							status = append(status, doc.GetNodeText(statusItemNode.Child(1)))
							isBlockSequence = true
						}
					})
					var statusRange protocol.Range
					// For multi-line arrays, include everything from after the colon to end of
					// multi-line value array.
					if isBlockSequence {
						keyRange := doc.NodeToRange(key)
						valueRange := doc.NodeToRange(value)
						statusRange = protocol.Range{
							Start: protocol.Position{
								Line:      keyRange.End.Line,
								Character: keyRange.End.Character + 1, // +1 to skip the ':'
							},
							End: valueRange.End,
						}
					} else {
						// For inline arrays, use the actual sequence node, not the value node
						// which may include an anchor. e.g.
						//
						//  requires:
						//    - job-name: &job-name-requires [ terminal statuses ]
						//                                   ^^^^^^^^^^^^^^^^^^^^^
						statusRange = doc.NodeToRange(statusesNode)
					}
					res = append(res, ast.Require{
						Name:        doc.GetNodeText(key),
						Status:      status,
						Range:       doc.NodeToRange(key),
						StatusRange: statusRange,
					})
				}
			}
		}
	})

	return res
}

func (doc *YamlDocument) parseMatrixAttributes(node *sitter.Node) (map[string][]ast.ParameterValue, string) {
	// node is a block_node
	blockMapping := GetChildOfType(node, "block_mapping")
	res := make(map[string][]ast.ParameterValue)
	alias := ""

	doc.iterateOnBlockMapping(blockMapping, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
		if keyNode == nil || valueNode == nil {
			return
		}
		keyName := doc.GetNodeText(keyNode)
		switch keyName {
		case "parameters":
			res = doc.parseMatrixParam(valueNode)
		case "alias":
			alias = doc.GetNodeText(valueNode)
		case "exclude":
		}
	})

	return res, alias
}

func (doc *YamlDocument) parseMatrixParam(node *sitter.Node) map[string][]ast.ParameterValue {
	// node is a block_node
	blockMapping := GetChildOfType(node, "block_mapping")
	res := make(map[string][]ast.ParameterValue)

	doc.iterateOnBlockMapping(blockMapping, func(child *sitter.Node) {
		keyNode := child.ChildByFieldName("key")
		if keyNode == nil {
			return
		}
		keyName := doc.GetNodeText(keyNode)

		paramValue, err := doc.parseParameterValue(child)
		if err != nil {
			return
		}
		res[keyName] = append(res[keyName], paramValue)

	})

	return res
}

func (doc *YamlDocument) buildJobsDAG(jobInvocations []ast.JobInvocation) map[string][]string {
	res := make(map[string][]string)
	for _, jobInvocation := range jobInvocations {
		for _, requirement := range jobInvocation.Requires {
			res[requirement.Name] = append(res[requirement.Name], jobInvocation.StepName)
		}
	}
	return res
}
