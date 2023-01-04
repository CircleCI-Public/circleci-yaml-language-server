package parser

import (
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

func (doc *YamlDocument) parseWorkflows(workflowsNode *sitter.Node) {
	// workflowsNode is of type block_node
	blockMappingNode := GetChildOfType(workflowsNode, "block_mapping")

	doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyName := doc.GetNodeText(child.ChildByFieldName("key"))

		switch keyName {
		case "version":
			if doc.Version >= 2.1 {
				rng := NodeToRange(child)
				doc.addDiagnostic(
					utils.CreateDiagnosticFromRange(
						rng,
						protocol.DiagnosticSeverityWarning,
						"Version key is deprecated since 2.1",
						[]protocol.CodeAction{
							utils.CreateCodeActionTextEdit("Delete version key", doc.URI,
								[]protocol.TextEdit{
									{
										Range:   rng,
										NewText: "",
									},
								}, false),
						},
					),
				)
			}
		default:
			workflow := doc.parseSingleWorkflow(child)
			if definedWorkflow, ok := doc.Workflows[workflow.Name]; ok {
				doc.addDiagnostic(protocol.Diagnostic{
					Severity: protocol.DiagnosticSeverityWarning,
					Range:    workflow.NameRange,
					Message:  "Workflow already defined",
					Source:   "cci-language-server",
				})
				doc.addDiagnostic(protocol.Diagnostic{
					Severity: protocol.DiagnosticSeverityWarning,
					Range:    definedWorkflow.NameRange,
					Message:  "Workflow already defined",
					Source:   "cci-language-server",
				})
				return
			}

			doc.Workflows[workflow.Name] = workflow
		}

	})
}

func (doc *YamlDocument) parseSingleWorkflow(workflowNode *sitter.Node) ast.Workflow {
	// workflowNode is a block_mapping_pair
	keyNode, valueNode := doc.GetKeyValueNodes(workflowNode)
	if keyNode == nil || valueNode == nil {
		return ast.Workflow{Range: NodeToRange(workflowNode)}
	}
	workflowName := doc.GetNodeText(keyNode)

	blockMappingNode := GetChildOfType(valueNode, "block_mapping")
	res := ast.Workflow{Range: NodeToRange(workflowNode), Name: workflowName, NameRange: NodeToRange(keyNode)}
	doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
		if keyNode == nil || valueNode == nil {
			return
		}
		keyName := doc.GetNodeText(keyNode)
		switch keyName {
		case "jobs":
			res.JobRefs = doc.parseJobReferences(valueNode)
			res.JobsDAG = doc.buildJobsDAG(res.JobRefs)
		case "triggers":
			res.HasTrigger = true
		}
	})

	return res
}

func (doc *YamlDocument) buildJobsDAG(jobRefs []ast.JobRef) map[string][]string {
	res := make(map[string][]string)
	for _, jobRef := range jobRefs {
		for _, requirement := range jobRef.Requires {
			res[requirement.Name] = append(res[requirement.Name], jobRef.StepName)
		}
	}
	return res
}

func (doc *YamlDocument) parseJobReferences(jobsRefsNode *sitter.Node) []ast.JobRef {
	// jobsRefsNode is block_node
	blockSequenceNode := GetChildOfType(jobsRefsNode, "block_sequence")
	jobReferences := []ast.JobRef{}
	if blockSequenceNode == nil {
		return jobReferences
	}

	iterateOnBlockSequence(blockSequenceNode, func(child *sitter.Node) {
		jobReferences = append(jobReferences, doc.parseSingleJobReference(child))
	})
	return jobReferences
}

func (doc *YamlDocument) parseSingleJobReference(jobRefNode *sitter.Node) ast.JobRef {
	res := ast.JobRef{MatrixParams: make(map[string][]ast.ParameterValue), Parameters: make(map[string]ast.ParameterValue)}
	if jobRefNode == nil {
		return res
	}
	res.JobRefRange = NodeToRange(jobRefNode)
	if jobRefNode.Type() != "block_sequence_item" {
		return res
	}

	if jobRefNode.ChildCount() == 1 {
		res.JobNameRange = protocol.Range{
			Start: protocol.Position{
				Line:      jobRefNode.StartPoint().Row,
				Character: jobRefNode.StartPoint().Column + 1,
			},
			End: protocol.Position{
				Line:      jobRefNode.StartPoint().Row,
				Character: jobRefNode.StartPoint().Column + 2,
			},
		}
		return res
	}

	element := jobRefNode.Child(1) // element is either flow_node or block_node
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
		res.JobNameRange = NodeToRange(element)
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

		res.JobNameRange = NodeToRange(key)
		res.JobName = doc.GetNodeText(key)
		res.StepNameRange = NodeToRange(key)
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
					res.TypeRange = NodeToRange(valueNode)
				case "requires":
					res.Requires = doc.parseSingleJobRequires(valueNode)
				case "name":
					res.StepNameRange = NodeToRange(valueNode)
					res.StepName = doc.GetNodeText(valueNode)
				case "context":
				case "filters":
				case "branches":
				case "tags":
				case "matrix":
					res.HasMatrix = true
					res.MatrixParams = doc.parseMatrixAttributes(valueNode)

				case "pre-steps":
					res.PreStepsRange = NodeToRange(child)
					res.PreSteps = doc.parseSteps(valueNode)

				case "post-steps":
					res.PostStepsRange = NodeToRange(child)
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

func (doc *YamlDocument) parseSingleJobRequires(node *sitter.Node) []ast.Require {
	array := doc.getNodeTextArrayWithRange(node)
	res := []ast.Require{}
	for _, require := range array {
		res = append(res, ast.Require{Name: require.Text, Range: require.Range})
	}
	return res
}

func (doc *YamlDocument) parseMatrixAttributes(node *sitter.Node) map[string][]ast.ParameterValue {
	// node is a block_node
	blockMapping := GetChildOfType(node, "block_mapping")
	res := make(map[string][]ast.ParameterValue)

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
		case "exclude":
		}
	})

	return res
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
