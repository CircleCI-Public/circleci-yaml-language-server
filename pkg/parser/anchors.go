package parser

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

func ParseYamlAnchors(doc *YamlDocument) map[string]YamlAnchor {
	rootNode := doc.RootNode

	// Mapping anchors
	anchorMap := map[string]YamlAnchor{}

	// Mapping all anchors
	ExecQuery(rootNode, "(anchor) @query", func(match *sitter.QueryMatch) {
		for _, capture := range match.Captures {
			node := capture.Node
			nameNode := GetChildOfType(node, "anchor_name")
			name := doc.GetNodeText(nameNode)
			valueNode := node.Parent()

			anchorMap[name] = YamlAnchor{
				DefinitionRange: doc.NodeToRange(node),
				References:      &[]protocol.Range{},
				ValueNode:       valueNode,
			}
		}
	})

	// Searching for all aliases
	ExecQuery(rootNode, "(alias) @query", func(match *sitter.QueryMatch) {
		for _, capture := range match.Captures {
			node := capture.Node
			name := doc.GetNodeText(node)[1:]

			aliasRange := doc.NodeToRange(node)
			anchor, ok := anchorMap[name]

			if !ok {
				continue
			}

			*anchor.References = append(*anchor.References, aliasRange)
		}
	})

	return anchorMap
}

func (doc *YamlDocument) IsYamlAliasPosition(pos protocol.Position) bool {
	for _, anchor := range doc.YamlAnchors {
		for _, aliasRange := range *anchor.References {
			if utils.PosInRange(aliasRange, pos) {
				return true
			}
		}
	}

	return false
}

func (doc *YamlDocument) GetYamlAnchorAtPosition(pos protocol.Position) (YamlAnchor, bool) {
	for _, anchor := range doc.YamlAnchors {
		if utils.PosInRange(anchor.DefinitionRange, pos) {
			return anchor, true
		}
	}

	return YamlAnchor{}, false
}
