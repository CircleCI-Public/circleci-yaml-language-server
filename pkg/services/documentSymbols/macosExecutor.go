package documentSymbols

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"go.lsp.dev/protocol"
)

func macosExecutorSymbols(macos ast.MacOSExecutor) protocol.DocumentSymbol {
	return protocol.DocumentSymbol{
		Name:           "xcode",
		Range:          macos.GetRange(),
		SelectionRange: macos.GetRange(),
		Detail:         macos.Xcode,
	}
}
