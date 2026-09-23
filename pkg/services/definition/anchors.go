package definition

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"go.lsp.dev/protocol"
)

func (def DefinitionStruct) searchAliasDefinition() []protocol.Location {
	pos := def.Params.Position

	if anchor, found := def.Doc.GetYamlAnchorAtPosition(pos); found {
		return []protocol.Location{{
			URI:   def.Params.TextDocument.URI,
			Range: anchor.DefinitionRange,
		}}
	}

	for _, anchor := range def.Doc.YamlAnchors {
		for _, aliasRange := range *anchor.References {
			if !position.InRange(aliasRange, pos) {
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
