package complete

import (
	"fmt"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func (ch *CompletionHandler) completeCommands() {
	command, err := findCommand(ch.Params.Position, ch.Doc)
	if err != nil {
		return
	}

	switch true {
	case position.InRange(command.ParametersRange, ch.Params.Position):
		ch.addParametersDefinitionCompletion(command.Parameters)
		return
	case position.InRange(command.StepsRange, ch.Params.Position):
		ch.completeSteps(command.Name, false, ch.nodeToComplete())
		return
	}

	if command.Description == "" {
		ch.addCompletionItemField("description")
	}
	if position.IsDefaultRange(command.ParametersRange) {
		ch.addCompletionItemField("parameters")
	}
	if len(command.Steps) == 0 {
		ch.addCompletionItemField("steps")
	}
}

func findCommand(pos protocol.Position, doc yamlparser.YamlDocument) (ast.Command, error) {
	for _, command := range doc.Commands {
		if position.InRange(command.Range, pos) {
			return command, nil
		}
	}
	return ast.Command{}, fmt.Errorf("no command found")
}

func (ch *CompletionHandler) userDefinedCommands() {
	for _, cmd := range ch.Doc.Commands {
		ch.addCompletionItem(cmd.Name)
	}
}

func (ch *CompletionHandler) orbCommands(nodeToComplete *sitter.Node) []protocol.CompletionItem {
	for _, orb := range ch.Doc.Orbs {
		orbInfo := ch.GetOrbInfo(orb)
		if orbInfo != nil {
			for cmdName := range orbInfo.Commands {
				cmdName = fmt.Sprintf("%s/%s", orb.Name, cmdName)

				if nodeToComplete == nil {
					ch.addCompletionItem(cmdName)
				} else {
					ch.addReplaceTextCompletionItem(nodeToComplete, cmdName)
				}
			}
		}
	}
	return ch.Items
}
