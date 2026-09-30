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

	executorsSymbol := sectionSymbol(
		document,
		"executors",
		document.ExecutorsRange,
		"Executors",
	)

	children := []protocol.DocumentSymbol{}

	for _, executor := range document.Executors {
		children = append(children, singleExecutorSymbols(executor))
	}
	children = append(children, aliasSymbols(document.Aliases.Executors, ExecutorSymbol)...)

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
		childrens = append(childrens, machineExecutorSymbols(executor, executor.Range))

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
		SelectionRange: selectionRange(executor.GetRange(), executor.GetNameRange()),
		Detail:         unlessZero(execType),
		Kind:           ExecutorSymbol,
		Children:       childrens,
	}

	return symbol
}

func envsSymbols(env ast2.Environment) protocol.DocumentSymbol {
	children := []protocol.DocumentSymbol{}

	for _, variable := range env.Variables {
		children = append(children, protocol.DocumentSymbol{
			Name:           variable.Name,
			Kind:           VariableSymbol,
			Range:          variable.Range,
			SelectionRange: variable.NameRange,
		})
	}

	return protocol.DocumentSymbol{
		Name:           "Environments",
		Kind:           ListSymbol,
		Range:          env.Range,
		SelectionRange: env.Range,
		Children:       children,
	}
}
