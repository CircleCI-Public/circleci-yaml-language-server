package parser

import (
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

var simpleOrbExistanceCache = make(map[string]bool)

func (doc *YamlDocument) GetOrbInfoFromName(name string, cache *utils.Cache) (*ast.OrbInfo, error) {
	// Searching within local orbs
	orbInfo, ok := doc.LocalOrbInfo[name]
	if ok {
		return orbInfo, nil
	}

	orb, ok := doc.Orbs[name]

	if !ok {
		return nil, nil
	}

	return doc.GetOrFetchOrbInfo(orb, cache)
}

func (doc *YamlDocument) GetOrFetchOrbInfo(orb ast.Orb, cache *utils.Cache) (*ast.OrbInfo, error) {
	// Searching within local orbs
	orbInfo, ok := doc.LocalOrbInfo[orb.Name]
	if ok {
		return orbInfo, nil
	}

	orbId := orb.Url.GetOrbID()

	// Searching within remote orbs
	orbInfo = cache.OrbCache.GetOrb(orbId)
	if orbInfo != nil {
		return orbInfo, nil
	}

	// Trying to fetch if not found
	var err error
	orbInfo, err = GetOrbInfo(orbId, cache, doc.Context)

	if err != nil {
		return &ast.OrbInfo{}, err
	}

	return orbInfo, nil
}

func (doc *YamlDocument) DoesOrbExist(orb ast.Orb, cache *utils.Cache) bool {
	lookup := orb.Url.Name
	exists, inMap := simpleOrbExistanceCache[lookup]

	if inMap {
		return exists
	}

	fetchedOrb, err := GetOrbByName(lookup, doc.Context)
	simpleOrbExistanceCache[lookup] = err == nil && fetchedOrb.Name != ""

	return simpleOrbExistanceCache[lookup]
}

func (doc *YamlDocument) parseOrbs(orbsNode *sitter.Node) {
	// orbsNode is a block_node
	blockMappingNode := GetChildMapping(orbsNode)
	if blockMappingNode == nil {
		return
	}

	doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		orb, localOrb := doc.parseSingleOrb(child)

		if orb != nil {
			doc.Orbs[orb.Name] = *orb
		}

		if localOrb != nil {
			doc.LocalOrbs = append(doc.LocalOrbs, *localOrb)
		}
	})
}

func (doc *YamlDocument) parseSingleOrb(orbNode *sitter.Node) (*ast.Orb, *LocalOrb) {
	// orbNode is a block_mapping_pair
	orbNameNode, orbContent := doc.GetKeyValueNodes(orbNode)
	orbName := doc.GetNodeText(orbNameNode)

	if orbContent == nil {
		return nil, nil
	}

	switch orbContent.Type() {
	case "flow_node":
		orbUrl := doc.getOrbURL(doc.GetNodeText(orbContent))
		orb := ast.Orb{
			Url:          orbUrl,
			Name:         orbName,
			Range:        doc.NodeToRange(orbNode),
			NameRange:    doc.NodeToRange(orbNameNode),
			VersionRange: doc.getOrbVersionRange(orbContent),
			ValueNode:    orbContent,
			ValueRange:   doc.NodeToRange(orbContent),
		}
		return &orb, nil

	case "block_node":
		localOrb, err := doc.parseLocalOrb(orbName, orbContent, doc.NodeToRange(orbNameNode).Start.Line)

		if err != nil {
			return nil, nil
		}

		orb := ast.Orb{
			Url: ast.OrbURL{
				Name:    orbName,
				Version: "",
				IsLocal: true,
			},
			Name:       orbName,
			Range:      doc.NodeToRange(orbNode),
			NameRange:  doc.NodeToRange(orbNameNode),
			ValueNode:  orbContent,
			ValueRange: doc.NodeToRange(orbContent),
		}

		return &orb, localOrb
	default:
		return nil, nil
	}
}

func (doc *YamlDocument) getOrbURL(orbUrl string) ast.OrbURL {
	splittedOrb := strings.Split((orbUrl), "@")

	if len(splittedOrb) > 1 {
		return ast.OrbURL{Name: splittedOrb[0], Version: splittedOrb[1], IsLocal: false}
	}

	return ast.OrbURL{Name: splittedOrb[0], Version: "volatile", IsLocal: false}
}

func (doc *YamlDocument) getOrbVersionRange(orbNode *sitter.Node) protocol.Range {
	orbNodeText := doc.GetRawNodeText(orbNode)
	orbRange := doc.NodeToRange(orbNode)
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
