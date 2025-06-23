package parser

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	sitter "github.com/smacker/go-tree-sitter"
)

func (doc *YamlDocument) parseRetention(retentionNode *sitter.Node) ast.RetentionSettings {
	res := ast.RetentionSettings{
		Range: doc.NodeToRange(retentionNode),
	}

	blockMapping := GetChildMapping(retentionNode)
	if blockMapping == nil {
		return res
	}

	doc.iterateOnBlockMapping(blockMapping, func(child *sitter.Node) {
		if child.Type() == "block_mapping_pair" || child.Type() == "flow_pair" {
			keyNode, valueNode := doc.GetKeyValueNodes(child)
			if keyNode == nil || valueNode == nil {
				return
			}

			keyName := doc.GetNodeText(keyNode)
			textAndRange := ast.TextAndRange{
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
