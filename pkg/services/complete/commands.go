package complete

import (
	"fmt"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

func (ch *CompletionHandler) completeCommands() {
	command, err := findCommand(ch.Params.Position, ch.Doc)
	if err != nil {
		return
	}

	switch true {
	case utils.PosInRange(command.ParametersRange, ch.Params.Position):
		ch.addParametersDefinitionCompletion(command.Parameters)
		return
	case utils.PosInRange(command.StepsRange, ch.Params.Position):
		nodeToComplete, _, _ := utils.NodeAtPos(ch.Doc.RootNode, ch.Params.Position)
		if nodeToComplete.Type() == ":" {
			nodeToComplete = nodeToComplete.PrevSibling()
		}
		ch.completeSteps(false, nodeToComplete)
		return
	}

	if command.Description == "" {
		ch.addCompletionItemField("description")
	}
	if command.Steps == nil || len(command.Steps) == 0 {
		ch.addCompletionItemField("steps")
	}
}

func findCommand(pos protocol.Position, doc yamlparser.YamlDocument) (ast.Command, error) {
	for _, command := range doc.Commands {
		if utils.PosInRange(command.Range, pos) {
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
