package documentSymbols

import (
	"fmt"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"go.lsp.dev/protocol"
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
			Detail:         fmt.Sprintf("%.1f", document.Version),
		},
	}
}
