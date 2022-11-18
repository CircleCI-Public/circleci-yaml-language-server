package parser

import (
	"strings"

	"github.com/circleci/circleci-yaml-language-server/pkg/ast"
	sitter "github.com/smacker/go-tree-sitter"
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
			Name:      orbName,
			Range:     NodeToRange(orbNode),
			NameRange: NodeToRange(orbNameNode),
			Url:       orbUrl,
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
