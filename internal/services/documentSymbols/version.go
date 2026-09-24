package documentSymbols

import (
	"fmt"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func resolveVersionSymbol(document *parser.YamlDocument) []protocol.DocumentSymbol {
	if position.IsDefaultRange(document.VersionRange) {
		return nil
	}

	return []protocol.DocumentSymbol{
		{
			Name:           "Version",
			Range:          document.VersionRange,
			SelectionRange: document.VersionRange,
			Detail:         unlessZero(fmt.Sprintf("%.1f", document.Version)),
		},
	}
}
