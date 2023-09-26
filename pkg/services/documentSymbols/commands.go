package documentSymbols

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func resolveCommandsSymbols(document *parser.YamlDocument) []protocol.DocumentSymbol {
	if utils.IsDefaultRange(document.CommandsRange) {
		return nil
	}

	commandsSymbols := symbolFromRange(
		document.CommandsRange,
		"Commands",
		ListSymbol,
	)

	children := []protocol.DocumentSymbol{}

	for _, command := range document.Commands {
		children = append(children, singleCommandSymbols(command))
	}

	commandsSymbols.Children = children

	return []protocol.DocumentSymbol{commandsSymbols}
}

func singleCommandSymbols(command ast.Command) protocol.DocumentSymbol {
	symbol := protocol.DocumentSymbol{
		Name:           command.Name,
		Range:          command.Range,
		SelectionRange: command.Range,
		Detail:         command.Description,
		Kind:           protocol.SymbolKind(CommandsSymbol),
	}

	if len(command.Steps) > 0 {
		stepChild := protocol.DocumentSymbol{
			Name:           "Steps",
			Range:          command.StepsRange,
			SelectionRange: command.StepsRange,
			Kind:           protocol.SymbolKind(ListSymbol),
			Children:       stepsSymbols(command.Steps),
		}

		symbol.Children = append(symbol.Children, stepChild)
	}

	if len(command.Parameters) > 0 {
		paramsChild := protocol.DocumentSymbol{
			Name:           "Parameters",
			Range:          command.StepsRange,
			SelectionRange: command.StepsRange,
			Kind:           protocol.SymbolKind(ListSymbol),
			Children:       parametersSymbols(command.Parameters),
		}

		symbol.Children = append(symbol.Children, paramsChild)
	}

	return symbol
}
