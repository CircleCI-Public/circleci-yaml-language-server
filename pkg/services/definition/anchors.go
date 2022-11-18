package definition

import (
	yamlparser "github.com/circleci/circleci-yaml-language-server/pkg/parser"
	"github.com/circleci/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func (def DefinitionStruct) searchAliasDefinition(doc yamlparser.YamlDocument) []protocol.Location {
	pos := def.Params.Position

	if anchor, found := doc.GetYamlAnchorAtPosition(pos); found {
		return []protocol.Location{{
			URI:   def.Params.TextDocument.URI,
			Range: anchor.DefinitionRange,
		}}
	}

	for _, anchor := range doc.YamlAnchors {
		for _, aliasRange := range *anchor.References {
			if !utils.PosInRange(aliasRange, pos) {
				continue
			}

			location := []protocol.Location{{
				URI:   def.Params.TextDocument.URI,
				Range: anchor.DefinitionRange,
			}}

			return location
		}
	}

	return nil
}
