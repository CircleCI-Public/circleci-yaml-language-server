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
				rng := doc.NodeToRange(child)
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
		return ast.Workflow{Range: doc.NodeToRange(workflowNode)}
	}

	workflowName := doc.GetNodeText(keyNode)

	blockMappingNode := GetChildOfType(valueNode, "block_mapping")
	res := ast.Workflow{Range: doc.NodeToRange(workflowNode), Name: workflowName, NameRange: doc.NodeToRange(keyNode)}

	doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)

		if keyNode == nil || valueNode == nil {
			return
		}

		keyName := doc.GetNodeText(keyNode)

		switch keyName {
		case "jobs":
			res.JobsRange = doc.NodeToRange(child)
			res.JobRefs = doc.parseJobReferences(valueNode)
			res.JobsDAG = doc.buildJobsDAG(res.JobRefs)
		case "triggers":
			res.HasTrigger = true
			res.TriggersRange = doc.NodeToRange(child)
			res.Triggers = doc.parseWorkflowTriggers(valueNode)
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
	blockSequenceNode := GetChildSequence(jobsRefsNode)
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
	res.JobRefRange = doc.NodeToRange(jobRefNode)
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
					matrixParams, alias := doc.parseMatrixAttributes(valueNode)
					res.MatrixParams = matrixParams
					if alias != "" {
						res.StepName = alias
					}
				case "serial-group":
					res.SerialGroup = doc.GetNodeText(valueNode)
					res.SerialGroupRange = doc.NodeToRange(valueNode)

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
				return ast.Require{Name: text, Status: defaultStatus, Range: doc.NodeToRange(anchorValueNode)}
			} else {
				return ast.Require{Name: doc.GetNodeText(node), Status: defaultStatus, Range: doc.NodeToRange(node)}
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
					res = append(res, ast.Require{Name: doc.GetNodeText(key), Status: status, Range: doc.NodeToRange(key)})
				} else {
					statusesNode := GetFirstChild(value)
					status := make([]string, 0, statusesNode.ChildCount())
					iterateOnBlockSequence(statusesNode, func(statusItemNode *sitter.Node) {
						if statusItemNode.Type() == "flow_node" {
							status = append(status, doc.GetNodeText(statusItemNode))
						}
						if statusItemNode.Type() == "block_sequence_item" {
							status = append(status, doc.GetNodeText(statusItemNode.Child(1)))
						}
					})
					res = append(res, ast.Require{Name: doc.GetNodeText(key), Status: status, Range: doc.NodeToRange(key)})
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

func (doc *YamlDocument) parseWorkflowTriggers(triggersNode *sitter.Node) []ast.WorkflowTrigger {
	triggers := []ast.WorkflowTrigger{}

	if triggersNode == nil || triggersNode.Type() != "block_node" {
		return triggers
	}

	sequence := triggersNode.Child(0)

	if sequence == nil || sequence.Type() != "block_sequence" {
		return triggers
	}

	iterateOnBlockSequence(sequence, func(seqItem *sitter.Node) {
		// For each item, try to guess the typeuru
		blockNode := seqItem.Child(0)

		if blockNode == nil {
			return
		}

		if blockNode.Type() == "-" {
			blockNode = blockNode.NextSibling()
		}

		trigger := doc.parseSingleWorkflowTrigger(blockNode)

		if trigger != nil {
			triggers = append(triggers, *trigger)
		}
	})

	return triggers
}

func (doc *YamlDocument) parseSingleWorkflowTrigger(node *sitter.Node) *ast.WorkflowTrigger {
	if node == nil || node.Type() != "block_node" {
		return nil
	}

	child := node.Child(0)

	if child == nil {
		return nil
	}

	if child.Type() == "-" {
		child = child.NextSibling()
	}

	if child == nil || child.Type() != "block_mapping" {
		return nil
	}

	child = child.Child(0)

	if child == nil || child.Type() != "block_mapping_pair" {
		return nil
	}

	nameNode, valueNode := doc.GetKeyValueNodes(child)

	if doc.GetNodeText(nameNode) == "schedule" {
		schedule := doc.parseSingleScheduleTrigger(valueNode)

		if schedule == nil {
			return nil
		}

		return &ast.WorkflowTrigger{
			Schedule: *schedule,
			Range:    doc.NodeToRange(valueNode),
		}
	}

	return nil
}

func (doc *YamlDocument) parseSingleScheduleTrigger(node *sitter.Node) *ast.ScheduleTrigger {
	if node == nil || node.Type() != "block_node" {
		return nil
	}

	blockMapping := node.Child(0)

	if blockMapping.Type() != "block_mapping" {
		return nil
	}

	crontab := ""
	filters := ast.WorkflowFilters{}

	// Iterate on the blockmapping keys & fill stuff in, bruh.

	doc.iterateOnBlockMapping(blockMapping, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
		key := doc.GetNodeText(keyNode)

		if key == "cron" {
			crontab = doc.GetNodeText(valueNode)
		} else if key == "filters" {
			f := doc.parseFilters(valueNode)

			if f != nil {
				filters = *f
			}
		}
	})

	// Cool, now construct the thingy thing
	scheduleTrigger := ast.ScheduleTrigger{
		Cron:    crontab,
		Filters: filters,
		Range:   doc.NodeToRange(blockMapping),
	}

	return &scheduleTrigger
}

func (doc *YamlDocument) parseFilters(node *sitter.Node) *ast.WorkflowFilters {
	if node == nil || node.Type() != "block_node" {
		return nil
	}

	blockMapping := node.Child(0)

	if blockMapping.Type() != "block_mapping" {
		return nil
	}

	filters := ast.WorkflowFilters{
		Range: doc.NodeToRange(blockMapping),
	}

	doc.iterateOnBlockMapping(blockMapping, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
		key := doc.GetNodeText(keyNode)

		if key == "branches" {
			b := doc.parseBranchFilter(valueNode)

			if b != nil {
				filters.Branches = *b
			}
		}
	})

	return &filters
}

func (doc *YamlDocument) parseBranchFilter(node *sitter.Node) *ast.BranchesFilter {
	if node == nil || node.Type() != "block_node" {
		return nil
	}

	blockMapping := node.Child(0)

	if blockMapping.Type() != "block_mapping" {
		return nil
	}

	branchesFilter := ast.BranchesFilter{
		Range: doc.NodeToRange(blockMapping),
	}

	doc.iterateOnBlockMapping(blockMapping, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
		key := doc.GetNodeText(keyNode)

		if valueNode == nil {
			return
		}

		if key == "only" {
			if valueNode.Child(0) != nil {
				branchesFilter.Only = doc.sequenceToStrings(valueNode.Child(0))
			}

			branchesFilter.OnlyRange = doc.NodeToRange(valueNode)
		} else if key == "ignore" {
			if valueNode.Child(0) != nil {
				branchesFilter.Ignore = doc.sequenceToStrings(valueNode.Child(0))
			}

			branchesFilter.IgnoreRange = doc.NodeToRange(valueNode)
		}
	})

	return &branchesFilter
}

func (doc *YamlDocument) sequenceToStrings(node *sitter.Node) []string {
	strs := []string{}

	if node == nil || (node.Type() != "block_sequence" && node.Type() != "flow_sequence") {
		return strs
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)

		if child.Type() == "block_sequence_item" {
			child = child.Child(0)

			if child.Type() == "-" {
				child = child.NextSibling()
			}
		} else if child.Type() != "flow_node" {
			continue
		}

		txt := doc.GetNodeText(child)

		strs = append(
			strs,
			txt,
		)
	}

	return strs
}
