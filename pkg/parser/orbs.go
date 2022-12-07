package parser

import (
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

func (doc *YamlDocument) parseOrbs(orbsNode *sitter.Node) {
	// orbsNode is a block_node
	blockMappingNode := GetChildMapping(orbsNode)
	if blockMappingNode == nil {
		return
	}
	iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		orb := doc.parseSingleOrb(child)
		if orb != nil {
			doc.Orbs[orb.Name] = *orb
		}
	})
}

func (doc *YamlDocument) parseSingleOrb(orbNode *sitter.Node) *ast.Orb {
	// orbNode is a block_mapping_pair
	orbNameNode := orbNode.ChildByFieldName("key")
	orbName := doc.GetNodeText(orbNameNode)
	orbContent := orbNode.ChildByFieldName("value")
	if orbContent == nil {
		return nil
	}
	switch orbContent.Type() {
	case "flow_node":
		orbUrl := doc.getOrbURL(doc.GetNodeText(orbContent))
		orb := ast.Orb{
			Url:          orbUrl,
			Name:         orbName,
			Range:        NodeToRange(orbNode),
			NameRange:    NodeToRange(orbNameNode),
			VersionRange: doc.getOrbVersionRange(orbContent),
		}
		return &orb
	case "block_node":
		doc.parseLocalOrb(orbName, orbContent)
		return nil
	default:
		return nil
	}
}

func (doc *YamlDocument) getOrbURL(orbUrl string) ast.OrbURL {
	splittedOrb := strings.Split((orbUrl), "@")

	if len(splittedOrb) > 1 {
		return ast.OrbURL{Name: splittedOrb[0], Version: splittedOrb[1]}
	}

	return ast.OrbURL{Name: splittedOrb[0], Version: "volatile"}
}

func (doc *YamlDocument) getOrbVersionRange(orbNode *sitter.Node) protocol.Range {
	orbNodeText := doc.GetRawNodeText(orbNode)
	orbRange := NodeToRange(orbNode)
	atIndex := strings.Index(orbNodeText, "@")
	if atIndex == -1 {
		return protocol.Range{}
	}
	return protocol.Range{
		Start: protocol.Position{
			Line:      orbRange.Start.Line,
			Character: orbRange.Start.Character + uint32(atIndex) + 1,
		},
		End: orbRange.End,
	}
}
