package documentSymbols

import (
	"go.lsp.dev/protocol"

	ast2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func resolveExecutorsSymbols(document *parser.YamlDocument) []protocol.DocumentSymbol {
	if position.IsDefaultRange(document.ExecutorsRange) {
		return nil
	}

	executorsSymbol := symbolFromRange(
		document.ExecutorsRange,
		"Executors",
		ListSymbol,
	)

	children := []protocol.DocumentSymbol{}

	for _, executor := range document.Executors {
		children = append(children, singleExecutorSymbols(executor))
	}

	executorsSymbol.Children = children

	return []protocol.DocumentSymbol{executorsSymbol}
}

func singleExecutorSymbols(executor ast2.Executor) protocol.DocumentSymbol {
	execType := ""
	childrens := []protocol.DocumentSymbol{}

	// TODO: More details on executors
	// -- little pickle when we have multiple types defined (Docker & Machine for example)

	switch executor := executor.(type) {
	case ast2.DockerExecutor:
		execType = "Docker"
		childrens = append(childrens, dockerExecutorSymbols(executor))

	case ast2.MachineExecutor:
		execType = "Machine"
		childrens = append(childrens, machineExecutorSymbols(executor))

	case ast2.MacOSExecutor:
		execType = "Mac OS"
		childrens = append(childrens, macosExecutorSymbols(executor))
	}

	envs := executor.GetEnvs()

	if !position.IsDefaultRange(envs.Range) {
		childrens = append(childrens, envsSymbols(envs))
	}

	symbol := protocol.DocumentSymbol{
		Name:           executor.GetName(),
		Range:          executor.GetRange(),
		SelectionRange: executor.GetRange(),
		Detail:         execType,
		Kind:           protocol.SymbolKind(ExecutorsSymbol),
		Children:       childrens,
	}

	return symbol
}

func envsSymbols(env ast2.Environment) protocol.DocumentSymbol {
	children := []protocol.DocumentSymbol{}

	for _, key := range env.Keys {
		children = append(children, protocol.DocumentSymbol{
			Name:           key,
			Range:          env.Range,
			SelectionRange: env.Range,
		})
	}

	return protocol.DocumentSymbol{
		Name:           "Environments",
		Range:          env.Range,
		SelectionRange: env.Range,
		Children:       children,
	}
}
