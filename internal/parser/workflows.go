package parser

import (
	"strconv"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/codeaction"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/paramref"
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
					diagnostic.New(
						rng,
						protocol.DiagnosticSeverityWarning,
						"Version key is deprecated since 2.1",
						[]protocol.CodeAction{
							codeaction.TextEdit("Delete version key", doc.URI,
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
					Message:  protocol.String("Workflow already defined"),
					Source:   protocol.NewOptional("cci-language-server"),
				})
				doc.addDiagnostic(protocol.Diagnostic{
					Severity: protocol.DiagnosticSeverityWarning,
					Range:    definedWorkflow.NameRange,
					Message:  protocol.String("Workflow already defined"),
					Source:   protocol.NewOptional("cci-language-server"),
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
			res.JobInvocations = doc.parseJobInvocations(valueNode)
			res.JobsDAG = doc.buildJobsDAG(res.JobInvocations)
		case "triggers":
			res.HasTrigger = true
			res.TriggersRange = doc.NodeToRange(child)
			res.Triggers = doc.parseWorkflowTriggers(valueNode)
		case "max_auto_reruns":
			res.MaxAutoRerunsRange = doc.NodeToRange(valueNode)
			valueText := doc.GetNodeText(valueNode)
			// A reference's value is only known once the config is compiled.
			res.HasMaxAutoReruns = !paramref.ContainsReference(valueText)
			if valueText != "" {
				if maxAutoReruns, err := strconv.Atoi(valueText); err == nil {
					res.MaxAutoReruns = maxAutoReruns
				}
			}
		}
	})

	return res
}

func (doc *YamlDocument) parseWorkflowTriggers(triggersNode *sitter.Node) []ast.WorkflowTrigger {
	triggers := []ast.WorkflowTrigger{}

	if triggersNode == nil || triggersNode.Kind() != "block_node" {
		return triggers
	}

	sequence := triggersNode.Child(0)

	if sequence == nil || sequence.Kind() != "block_sequence" {
		return triggers
	}

	iterateOnBlockSequence(sequence, func(seqItem *sitter.Node) {
		// For each item, try to guess the typeuru
		blockNode := seqItem.Child(0)

		if blockNode == nil {
			return
		}

		if blockNode.Kind() == "-" {
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
	if node == nil || node.Kind() != "block_node" {
		return nil
	}

	child := node.Child(0)

	if child == nil {
		return nil
	}

	if child.Kind() == "-" {
		child = child.NextSibling()
	}

	if child == nil || child.Kind() != "block_mapping" {
		return nil
	}

	child = child.Child(0)

	if child == nil || child.Kind() != "block_mapping_pair" {
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
	if node == nil || node.Kind() != "block_node" {
		return nil
	}

	blockMapping := node.Child(0)

	if blockMapping.Kind() != "block_mapping" {
		return nil
	}

	crontab := ""
	filters := ast.WorkflowFilters{}

	// Iterate on the blockmapping keys & fill stuff in, bruh.

	doc.iterateOnBlockMapping(blockMapping, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
		key := doc.GetNodeText(keyNode)

		switch key {
		case "cron":
			crontab = doc.GetNodeText(valueNode)
		case "filters":
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
	if node == nil || node.Kind() != "block_node" {
		return nil
	}

	blockMapping := node.Child(0)

	if blockMapping.Kind() != "block_mapping" {
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
	if node == nil || node.Kind() != "block_node" {
		return nil
	}

	blockMapping := node.Child(0)

	if blockMapping.Kind() != "block_mapping" {
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

		switch key {
		case "only":
			if valueNode.Child(0) != nil {
				branchesFilter.Only = doc.sequenceToStrings(valueNode.Child(0))
			}

			branchesFilter.OnlyRange = doc.NodeToRange(valueNode)
		case "ignore":
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

	if node == nil || (node.Kind() != "block_sequence" && node.Kind() != "flow_sequence") {
		return strs
	}

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		if child.Kind() == "block_sequence_item" {
			child = child.Child(0)

			if child.Kind() == "-" {
				child = child.NextSibling()
			}
		} else if child.Kind() != "flow_node" {
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
