package parser

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	ast2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
)

func (doc *YamlDocument) parseRetention(retentionNode *sitter.Node) ast2.RetentionSettings {
	res := ast2.RetentionSettings{
		Range: doc.NodeToRange(retentionNode),
	}

	blockMapping := GetChildMapping(retentionNode)
	if blockMapping == nil {
		return res
	}

	doc.iterateOnBlockMapping(blockMapping, func(child *sitter.Node) {
		if child.Kind() == "block_mapping_pair" || child.Kind() == "flow_pair" {
			keyNode, valueNode := doc.GetKeyValueNodes(child)
			if keyNode == nil || valueNode == nil {
				return
			}

			keyName := doc.GetNodeText(keyNode)
			textAndRange := ast2.TextAndRange{
				Text:  doc.GetNodeText(valueNode),
				Range: doc.NodeToRange(child),
			}
			switch keyName {
			case "caches":
				res.Caches = textAndRange
			}
		}
	})

	return res
}
