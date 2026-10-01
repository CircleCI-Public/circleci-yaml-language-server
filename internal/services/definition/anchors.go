package definition

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func (def DefinitionStruct) searchAliasDefinition() []Link {
	pos := def.Params.Position

	if anchor, found := def.Doc.GetYamlAnchorAtPosition(pos); found {
		return []Link{{
			Origin: anchor.DefinitionRange,
			URI:    def.Params.TextDocument.URI,
			Range:  anchor.DefinitionRange,
		}}
	}

	for _, anchor := range def.Doc.YamlAnchors {
		for _, aliasRange := range *anchor.References {
			if !position.InRange(aliasRange, pos) {
				continue
			}

			return []Link{{
				Origin: aliasRange,
				URI:    def.Params.TextDocument.URI,
				Range:  anchor.DefinitionRange,
			}}
		}
	}

	return nil
}
