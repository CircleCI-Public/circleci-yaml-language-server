package documentSymbols

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"go.lsp.dev/protocol"
)

func resolveSetupSymbol(doc *parser.YamlDocument) (symbols []protocol.DocumentSymbol) {
	if doc.Setup {
		symbols = append(symbols, protocol.DocumentSymbol{
			Name:           "setup: true",
			Range:          doc.SetupRange,
			SelectionRange: doc.SetupRange,
			Kind:           protocol.SymbolKindConstant,
			Detail:         "Continuation workflow enabled",
		})
	}
	return
}
