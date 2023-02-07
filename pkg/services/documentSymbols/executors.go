package documentSymbols

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func resolveExecutorsSymbols(document *parser.YamlDocument) []protocol.DocumentSymbol {
	if utils.IsDefaultRange(document.ExecutorsRange) {
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

func singleExecutorSymbols(executor ast.Executor) protocol.DocumentSymbol {
	execType := ""
	childrens := []protocol.DocumentSymbol{}

	// TODO: More details on executors
	// -- little pickle when we have multiple types defined (Docker & Machine for example)

	switch executor.(type) {
	case ast.DockerExecutor:
		execType = "Docker"
		childrens = append(childrens, dockerExecutorSymbols(executor.(ast.DockerExecutor)))

	case ast.MachineExecutor:
		execType = "Machine"
		childrens = append(childrens, machineExecutorSymbols(executor.(ast.MachineExecutor)))

	case ast.MacOSExecutor:
		execType = "Mac OS"
		childrens = append(childrens, macosExecutorSymbols(executor.(ast.MacOSExecutor)))
	}

	envs := executor.GetEnvs()

	if !utils.IsDefaultRange(envs.Range) {
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

func envsSymbols(env ast.Environment) protocol.DocumentSymbol {
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
