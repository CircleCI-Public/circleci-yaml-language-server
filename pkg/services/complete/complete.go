package complete

import (
	"fmt"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

type CompletionHandler struct {
	Params protocol.CompletionParams

	Doc     yamlparser.YamlDocument
	DocTag  string
	DocDiff string

	Items []protocol.CompletionItem
	Cache *utils.Cache
}

func (ch *CompletionHandler) GetCompletionItems() {
	node, _, err := utils.NodeAtPos(ch.Doc.RootNode, ch.Params.Position)
	if err == nil {
		ch.addParameterReferenceCompletion(node)
		if len(ch.Items) > 0 {
			return
		}
	}

	modifiedDocs := ch.Doc.ModifyTextForAutocomplete(ch.Params.Position)

	for _, doc := range modifiedDocs {
		ch.Doc = doc.Document
		ch.DocTag = doc.Tag
		ch.DocDiff = doc.Diff

		if ch.Doc.IsYamlAliasPosition(ch.Params.Position) {
			ch.completeAnchors()
		} else if utils.PosInRange(ch.Doc.WorkflowRange, ch.Params.Position) {
			ch.completeWorkflows()
		} else if utils.PosInRange(ch.Doc.JobsRange, ch.Params.Position) {
			ch.completeJobs()
		} else if utils.PosInRange(ch.Doc.CommandsRange, ch.Params.Position) {
			ch.completeCommands()
		} else if utils.PosInRange(ch.Doc.ExecutorsRange, ch.Params.Position) {
			ch.completeExecutors()
		}

		if len(ch.Items) > 0 {
			break
		}
	}
}

func (ch *CompletionHandler) addCompletionItem(label string) {
	ch.Items = append(ch.Items, protocol.CompletionItem{
		Label: label,
	})
}

func (ch *CompletionHandler) addReplaceTextCompletionItem(node *sitter.Node, newText string) {
	ch.Items = append(ch.Items, protocol.CompletionItem{
		Label: newText,
		TextEdit: &protocol.TextEdit{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      node.StartPoint().Row,
					Character: node.StartPoint().Column,
				},
				End: protocol.Position{
					Line:      node.EndPoint().Row,
					Character: node.EndPoint().Column,
				},
			},
			NewText: newText,
		},
	})
}

func (ch *CompletionHandler) addCompletionItemField(label string) {
	ch.addCompletionItemFieldWithCustomText(label, ": ")
}

func (ch *CompletionHandler) addCompletionItemFieldWithNewLine(label string) {
	ch.addCompletionItemFieldWithCustomText(label, ": \n\t")
}

func (ch *CompletionHandler) addCompletionItemFieldWithCustomText(label string, customText string) {
	ch.Items = append(ch.Items, protocol.CompletionItem{
		Label:      label,
		InsertText: fmt.Sprintf("%s%s", label, customText),
	})
}

func (ch *CompletionHandler) GetOrbInfo(orb ast.Orb) *ast.OrbInfo {
	return ch.Doc.GetOrbInfo(ch.Cache, orb.Name)
}
