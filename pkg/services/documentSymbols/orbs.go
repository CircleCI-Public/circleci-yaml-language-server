package documentSymbols

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func resolveOrbSymbols(document *parser.YamlDocument) []protocol.DocumentSymbol {
	if utils.IsDefaultRange(document.OrbsRange) {
		return nil
	}

	symbol := symbolFromRange(
		document.OrbsRange,
		"Orbs",
		ListSymbol,
	)

	children := []protocol.DocumentSymbol{}

	for _, orb := range document.Orbs {
		children = append(
			children,
			protocol.DocumentSymbol{
				Name:           orb.Name,
				Kind:           protocol.SymbolKind(OrbSymbol),
				Range:          orb.Range,
				SelectionRange: orb.Range,
				Detail:         orb.Url.Version,
			},
		)
	}

	symbol.Children = children

	return []protocol.DocumentSymbol{symbol}
}

func symbolFromRange(rng protocol.Range, label string, symbol float64) protocol.DocumentSymbol {
	return protocol.DocumentSymbol{
		Name:           label,
		Kind:           protocol.SymbolKind(symbol),
		Range:          rng,
		SelectionRange: rng,
	}
}
