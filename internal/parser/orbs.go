package parser

import (
	"log/slog"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
)

func (doc *YamlDocument) GetOrbInfoFromName(name string, cache *cache.Cache) (*ast.OrbInfo, error) {
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

func (doc *YamlDocument) GetOrFetchOrbInfo(orb ast.Orb, cache *cache.Cache) (*ast.OrbInfo, error) {
	// Searching within local orbs
	orbInfo, ok := doc.LocalOrbInfo[orb.Name]
	if ok {
		return orbInfo, nil
	}

	// An orb at a URL that can't be fetched declares nothing that is known.
	if orb.Url.IsURL {
		orbInfo, err := GetURLOrbInfo(orb.Url.Name, cache, doc.Context)
		if orbInfo == nil {
			return &ast.OrbInfo{}, err
		}
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

// DoesOrbExist reports whether a remote orb exists and the token can see it.
// An orb that could not be looked up is reported as existing: that the lookup
// failed says nothing about the orb.
func (doc *YamlDocument) DoesOrbExist(orb ast.Orb, cache *cache.Cache) bool {
	found, err := cache.OrbPackages.Orb(doc.Context.OrbRegistry(), orb.Url.Name)
	if err != nil {
		slog.Warn("looking up orb", "orb", orb.Url.Name, "err", err)
		return true
	}

	return found != nil
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

	switch orbContent.Kind() {
	case "flow_node":
		if isEmptyFlowMapping(orbContent) {
			return doc.placeholderOrb(orbName, orbNode, orbNameNode, orbContent), nil
		}

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
		localOrb, err := doc.parseLocalOrb(orbName, orbContent)

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

// placeholderOrb is an orb declared as `{}`, which is filled in before the
// config is run, as orb-tools/continue does with the orb under test. It is
// taken as a local orb that declares nothing, so that nothing is fetched for
// it, and references into it are not checked.
func (doc *YamlDocument) placeholderOrb(orbName string, orbNode, orbNameNode, orbContent *sitter.Node) *ast.Orb {
	doc.LocalOrbInfo[orbName] = &ast.OrbInfo{
		IsLocal: true,
		OrbParsedAttributes: ast.OrbParsedAttributes{
			URI:       doc.URI,
			Name:      orbName,
			Commands:  map[string]ast.Command{},
			Jobs:      map[string]ast.Job{},
			Executors: map[string]ast.Executor{},
		},
	}

	return &ast.Orb{
		Url: ast.OrbURL{
			Name:    orbName,
			IsLocal: true,
		},
		Name:          orbName,
		Range:         doc.NodeToRange(orbNode),
		NameRange:     doc.NodeToRange(orbNameNode),
		ValueNode:     orbContent,
		ValueRange:    doc.NodeToRange(orbContent),
		IsPlaceholder: true,
	}
}

func isEmptyFlowMapping(flowNode *sitter.Node) bool {
	child := GetFirstChild(flowNode)
	return child != nil && child.Kind() == "flow_mapping" && child.NamedChildCount() == 0
}

func (doc *YamlDocument) getOrbURL(orbUrl string) ast.OrbURL {
	if isURLOrbReference(orbUrl) {
		return ast.OrbURL{Name: orbUrl, IsURL: true}
	}

	splittedOrb := strings.Split((orbUrl), "@")

	if len(splittedOrb) > 1 {
		return ast.OrbURL{Name: splittedOrb[0], Version: splittedOrb[1], IsLocal: false}
	}

	return ast.OrbURL{Name: splittedOrb[0], Version: "volatile", IsLocal: false}
}

// isURLOrbReference reports whether an orb is referenced by a URL rather than
// as namespace/name@version. The compiler takes any reference that parses as a
// URL, and only fetches it if the organization allows its prefix.
func isURLOrbReference(reference string) bool {
	return strings.Contains(reference, "://")
}

func (doc *YamlDocument) getOrbVersionRange(orbNode *sitter.Node) protocol.Range {
	orbNodeText := doc.GetRawNodeText(orbNode)
	if isURLOrbReference(orbNodeText) {
		return protocol.Range{}
	}
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

func (doc *YamlDocument) GetOrbURLDefinition(node *sitter.Node) ast.OrbURLDefinition {
	orbText := doc.GetNodeText(node)
	orbRange := doc.NodeToRange(node)
	return getOrbDefinitionFromTextAndRange(orbText, orbRange)
}

func getOrbDefinitionFromTextAndRange(orbText string, orbRange protocol.Range) (def ast.OrbURLDefinition) {
	//
	// Namespace
	//
	endOfNs := strings.Index(orbText, "/")
	if endOfNs == -1 {
		endOfNs = len(orbText)
	}

	def.Namespace.Text = orbText[:endOfNs]
	def.Namespace.Range.Start = orbRange.Start
	def.Namespace.Range.End = orbRange.Start
	def.Namespace.Range.End.Character += uint32(endOfNs)
	if endOfNs == len(orbText) {
		return
	}

	//
	// Name
	//
	endOfName := strings.Index(orbText, "@")
	if endOfName == -1 {
		endOfName = len(orbText)
	}

	def.Name.Text = orbText[endOfNs+1 : endOfName]
	def.Name.Range.Start = def.Namespace.Range.End
	def.Name.Range.Start.Character += 1
	def.Name.Range.End = orbRange.Start
	def.Name.Range.End.Character += uint32(endOfName)
	if endOfName == len(orbText) {
		return
	}

	//
	// Version
	//
	def.Version.Text = orbText[endOfName+1:]
	def.Version.Range.Start = def.Name.Range.End
	def.Version.Range.Start.Character += 1
	def.Version.Range.End = orbRange.End

	return
}
