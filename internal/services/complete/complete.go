package complete

import (
	"fmt"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
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
		} else if !ch.completeTopLevel() {
			ch.completeSection()
		}

		if len(ch.Items) > 0 {
			break
		}
	}
}

// completeSection completes in the top-level section the cursor is in.
func (ch *CompletionHandler) completeSection() {
	switch pos := ch.Params.Position; {
	case position.InRange(ch.Doc.WorkflowRange, pos):
		ch.completeWorkflows()
	case position.InRange(ch.Doc.JobsRange, pos):
		ch.completeJobs()
	case position.InRange(ch.Doc.JobGroupsRange, pos):
		ch.completeJobGroups()
	case position.InRange(ch.Doc.CommandsRange, pos):
		ch.completeCommands()
	case position.InRange(ch.Doc.ExecutorsRange, pos):
		ch.completeExecutors()
	case position.InRange(ch.Doc.OrbsRange, pos):
		if !ch.completeInInlineOrb() {
			ch.completeOrbs()
		}
	case position.InRange(ch.Doc.FunctionsRange, pos):
		ch.completeFunctionVersion()
	case position.InRange(ch.Doc.PipelineParametersRange, pos):
		ch.completeParameterDefinitions(ch.Doc.PipelineParameters, pipelineParameterTypes)
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
		Detail:   unlessEmpty(detail),
		SortText: unlessEmpty(sortText),
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
		InsertText: protocol.NewOptional(fmt.Sprintf("%s%s%s", beforeText, label, afterText)),
		Detail:     unlessEmpty(detail),
		SortText:   unlessEmpty(sortText),
	})
}

// unlessEmpty leaves an empty string out of the item, as it always has been,
// rather than sending it as "".
func unlessEmpty(s string) protocol.Optional[string] {
	if s == "" {
		return protocol.Optional[string]{}
	}
	return protocol.NewOptional(s)
}

func (ch *CompletionHandler) GetOrbInfo(orb ast.Orb) *ast.OrbInfo {
	orbInfo, _ := ch.Doc.GetOrFetchOrbInfo(orb, ch.Cache)
	return orbInfo
}
