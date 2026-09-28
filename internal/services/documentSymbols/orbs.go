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

	symbol := symbolFromRange(
		document.OrbsRange,
		"Orbs",
		SectionSymbol,
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

func symbolFromRange(rng protocol.Range, label string, kind protocol.SymbolKind) protocol.DocumentSymbol {
	return protocol.DocumentSymbol{
		Name:           label,
		Kind:           kind,
		Range:          rng,
		SelectionRange: rng,
	}
}
