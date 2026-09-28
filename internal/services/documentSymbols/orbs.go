package documentSymbols

import (
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func resolveOrbSymbols(document *parser.YamlDocument) []protocol.DocumentSymbol {
	if position.IsDefaultRange(document.OrbsRange) {
		return nil
	}

	symbol := sectionSymbol(
		document,
		"orbs",
		document.OrbsRange,
		"Orbs",
	)

	children := []protocol.DocumentSymbol{}

	for _, orb := range document.Orbs {
		children = append(
			children,
			protocol.DocumentSymbol{
				Name:           orb.Name,
				Kind:           OrbSymbol,
				Range:          orb.Range,
				SelectionRange: selectionRange(orb.Range, orb.NameRange),
				Detail:         unlessZero(orb.Url.Version),
			},
		)
	}

	symbol.Children = children

	return []protocol.DocumentSymbol{symbol}
}

// sectionSymbol is the symbol of a top-level section, from its key to the end
// of its value, rng, with the key selected.
func sectionSymbol(document *parser.YamlDocument, key string, rng protocol.Range, label string) protocol.DocumentSymbol {
	keyRange, ok := document.SectionKeyRanges[key]
	if !ok {
		return symbolFromRange(rng, label, SectionSymbol)
	}

	return protocol.DocumentSymbol{
		Name:           label,
		Kind:           SectionSymbol,
		Range:          protocol.Range{Start: keyRange.Start, End: rng.End},
		SelectionRange: keyRange,
	}
}

func symbolFromRange(rng protocol.Range, label string, kind protocol.SymbolKind) protocol.DocumentSymbol {
	return protocol.DocumentSymbol{
		Name:           label,
		Kind:           kind,
		Range:          rng,
		SelectionRange: rng,
	}
}
