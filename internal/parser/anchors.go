package parser

import (
	sitter "github.com/odvcencio/gotreesitter"
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/yamltree"
)

var (
	anchorsQuery = yamltree.MustCompileQuery("(anchor) @query")
	aliasesQuery = yamltree.MustCompileQuery("(alias) @query")
)

func ParseYamlAnchors(doc *YamlDocument) map[string]YamlAnchor {
	rootNode := doc.RootNode

	// Mapping anchors
	anchorMap := map[string]YamlAnchor{}

	// Mapping all anchors
	anchorsQuery.Run(rootNode, func(match *sitter.QueryMatch) {
		for _, capture := range match.Captures {
			node := capture.Node
			nameNode := GetChildOfType(node, "anchor_name")
			name := doc.GetNodeText(nameNode)
			valueNode := node.Parent()

			// A redefined anchor keeps its last definition, so an alias
			// written before the redefinition resolves to the wrong one. No
			// surveyed config does that.
			anchorMap[name] = YamlAnchor{
				DefinitionRange: doc.NodeToRange(node),
				References:      &[]protocol.Range{},
				ValueNode:       valueNode,
			}
		}
	})

	// Searching for all aliases
	aliasesQuery.Run(rootNode, func(match *sitter.QueryMatch) {
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
			if position.InRange(aliasRange, pos) {
				return true
			}
		}
	}

	return false
}

func (doc *YamlDocument) GetYamlAnchorAtPosition(pos protocol.Position) (YamlAnchor, bool) {
	for _, anchor := range doc.YamlAnchors {
		if position.InRange(anchor.DefinitionRange, pos) {
			return anchor, true
		}
	}

	return YamlAnchor{}, false
}
