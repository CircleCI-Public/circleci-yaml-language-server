package complete

import (
	"fmt"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"
)

type CompletionHandler struct {
	Params protocol.CompletionParams

	Doc     yamlparser.YamlDocument
	DocTag  string
	DocDiff string

	Items   []protocol.CompletionItem
	Cache   *cache.Cache
	Context *session.Settings
}

func (ch *CompletionHandler) GetCompletionItems() {
	node, _, err := position.NodeAt(ch.Doc.RootNode, ch.Params.Position)
	if err == nil {
		ch.addParameterReferenceCompletion(node)
		if len(ch.Items) > 0 {
			return
		}
	}

	modifiedDocs := ch.Doc.ModifyTextForAutocomplete(ch.Params.Position)
	// The variants are this handler's to close; the original belongs to
	// whoever parsed it.
	defer func() {
		for _, doc := range modifiedDocs {
			if doc.Tag != "original" {
				doc.Document.Close()
			}
		}
	}()

	for _, doc := range modifiedDocs {
		ch.Doc = doc.Document
		ch.DocTag = doc.Tag
		ch.DocDiff = doc.Diff

		if ch.Doc.IsYamlAliasPosition(ch.Params.Position) {
			ch.completeAnchors()
		} else if position.InRange(ch.Doc.WorkflowRange, ch.Params.Position) {
			ch.completeWorkflows()
		} else if position.InRange(ch.Doc.JobsRange, ch.Params.Position) {
			ch.completeJobs()
		} else if position.InRange(ch.Doc.JobGroupsRange, ch.Params.Position) {
			ch.completeJobGroups()
		} else if position.InRange(ch.Doc.CommandsRange, ch.Params.Position) {
			ch.completeCommands()
		} else if position.InRange(ch.Doc.ExecutorsRange, ch.Params.Position) {
			ch.completeExecutors()
		} else if position.InRange(ch.Doc.OrbsRange, ch.Params.Position) {
			ch.completeOrbs()
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

func (ch *CompletionHandler) addCompletionItemWithDetail(label string, detail string, sortText string) {
	ch.Items = append(ch.Items, protocol.CompletionItem{
		Label:    label,
		Detail:   detail,
		SortText: sortText,
	})
}

func (ch *CompletionHandler) addReplaceTextCompletionItem(node *sitter.Node, newText string) {
	ch.Items = append(ch.Items, protocol.CompletionItem{
		Label: newText,
		TextEdit: &protocol.TextEdit{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      position.Start(node).Line,
					Character: position.Start(node).Character,
				},
				End: protocol.Position{
					Line:      position.End(node).Line,
					Character: position.End(node).Character,
				},
			},
			NewText: newText,
		},
	})
}

func (ch *CompletionHandler) addCompletionItemField(label string) {
	ch.addCompletionItemFieldWithCustomText(label, "", ": ", "", "")
}

func (ch *CompletionHandler) addCompletionItemFieldWithNewLine(label string) {
	ch.addCompletionItemFieldWithCustomText(label, "", ": \n\t", "", "")
}

func (ch *CompletionHandler) addCompletionItemFieldWithCustomText(label string, beforeText string, afterText string, detail string, sortText string) {
	ch.Items = append(ch.Items, protocol.CompletionItem{
		Label:      label,
		InsertText: fmt.Sprintf("%s%s%s", beforeText, label, afterText),
		Detail:     detail,
		SortText:   sortText,
	})
}

func (ch *CompletionHandler) GetOrbInfo(orb ast.Orb) *ast.OrbInfo {
	orbInfo, _ := ch.Doc.GetOrFetchOrbInfo(orb, ch.Cache)
	return orbInfo
}
