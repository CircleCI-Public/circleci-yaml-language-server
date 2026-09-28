package documentSymbols

import (
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
)

// aliasSymbols are the symbols of a section's aliases, such as
// `build: orb/build`, of the same kind as what they name, with their target
// as the detail.
func aliasSymbols(aliases map[string]ast.Alias, kind protocol.SymbolKind) []protocol.DocumentSymbol {
	symbols := make([]protocol.DocumentSymbol, 0, len(aliases))
	for _, alias := range aliases {
		symbols = append(symbols, protocol.DocumentSymbol{
			Name:           alias.Name,
			Kind:           kind,
			Range:          alias.Range,
			SelectionRange: selectionRange(alias.Range, alias.NameRange),
			Detail:         unlessZero(alias.Target),
		})
	}
	return symbols
}
