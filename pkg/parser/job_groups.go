package parser

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	sitter "github.com/smacker/go-tree-sitter"
)

func (doc *YamlDocument) parseJobGroups(jobGroupsNode *sitter.Node) {
	blockMappingNode := GetChildMapping(jobGroupsNode)
	if blockMappingNode == nil {
		return
	}

	doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		jobGroup := doc.parseSingleJobGroup(child)
		doc.JobGroups[jobGroup.Name] = jobGroup
	})
}

func (doc *YamlDocument) parseSingleJobGroup(jobGroupNode *sitter.Node) ast.JobGroup {
	res := ast.JobGroup{}
	res.Range = doc.NodeToRange(jobGroupNode)

	jobGroupNameNode, valueNode := doc.GetKeyValueNodes(jobGroupNode)

	if jobGroupNameNode == nil || valueNode == nil {
		return res
	}

	jobGroupName := doc.GetNodeText(jobGroupNameNode)
	res.Name = doc.getAttributeName(jobGroupName)
	res.NameRange = doc.NodeToRange(jobGroupNameNode)

	blockMappingNode := GetChildMapping(valueNode)
	if blockMappingNode == nil {
		return res
	}

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
		}
	})

	return res
}
