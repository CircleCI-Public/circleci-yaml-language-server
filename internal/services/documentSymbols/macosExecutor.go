package documentSymbols

import (
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
)

func macosExecutorSymbols(macos ast.MacOSExecutor) protocol.DocumentSymbol {
	return protocol.DocumentSymbol{
		Name:           "xcode",
		Range:          macos.GetRange(),
		SelectionRange: macos.GetRange(),
		Detail:         unlessZero(macos.Xcode),
	}
}
