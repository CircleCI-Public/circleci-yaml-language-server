package documentSymbols

import (
	"strconv"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"go.lsp.dev/protocol"
)

func resolveSetupSymbol(doc *parser.YamlDocument) (symbols []protocol.DocumentSymbol) {
	if !position.IsDefaultRange(doc.SetupRange) {
		symbols = append(symbols, protocol.DocumentSymbol{
			Name:           "Setup",
			Kind:           protocol.SymbolKindBoolean,
			Detail:         strconv.FormatBool(doc.Setup),
			Range:          doc.SetupRange,
			SelectionRange: doc.SetupRange,
		})
	}
	return
}
