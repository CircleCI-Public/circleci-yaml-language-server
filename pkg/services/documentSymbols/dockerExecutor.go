package documentSymbols

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"go.lsp.dev/protocol"
)

func dockerExecutorSymbols(dockerExec ast.DockerExecutor) protocol.DocumentSymbol {
	symbol := protocol.DocumentSymbol{
		Name:           "Docker",
		Range:          dockerExec.Range,
		SelectionRange: dockerExec.Range,
		Kind:           protocol.SymbolKind(DockerSymbol),
	}

	for _, img := range dockerExec.Image {
		name := img.Name

		if name == "" {
			name = img.Image.FullPath
		}

		if name == "" {
			continue
		}

		symbol.Children = append(symbol.Children, protocol.DocumentSymbol{
			Name:           name,
			Range:          img.ImageRange,
			SelectionRange: img.ImageRange,
			Kind:           8,
		})
	}

	return symbol
}
