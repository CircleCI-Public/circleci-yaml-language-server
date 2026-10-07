package complete

import (
	"context"
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

func (ch *CompletionHandler) GetCompletionItems(ctx context.Context) {
	ch.completePipelineValues()
	if len(ch.Items) > 0 {
		return
	}

	ch.completeParameterReferences()
	if len(ch.Items) > 0 {
		return
	}

	// The copies are this handler's to close; the original belongs to
	// whoever parsed it. ch.Doc is read until the handler returns, so the
	// copies are closed then.
	var copies []yamlparser.YamlDocument
	defer func() {
		for _, doc := range copies {
			doc.Close()
		}
	}()

	for doc := range ch.Doc.ModifyTextForAutocomplete(ch.Params.Position) {
		if doc.Tag != "original" {
			copies = append(copies, doc.Document)
		}
		ch.Doc = doc.Document
		ch.DocTag = doc.Tag
		ch.DocDiff = doc.Diff

		if ch.Doc.IsYamlAliasPosition(ch.Params.Position) {
			ch.completeAnchors()
		} else if !ch.completeTopLevel() {
			ch.completeSection(ctx)
		}

		if len(ch.Items) > 0 {
			break
		}
	}
}

// completeSection completes in the top-level section the cursor is in.
func (ch *CompletionHandler) completeSection(ctx context.Context) {
	switch pos := ch.Params.Position; {
	case position.InRange(ch.Doc.WorkflowRange, pos):
		ch.completeWorkflows(ctx)
	case position.InRange(ch.Doc.JobsRange, pos):
		ch.completeJobs(ctx)
	case position.InRange(ch.Doc.JobGroupsRange, pos):
		ch.completeJobGroups(ctx)
	case position.InRange(ch.Doc.CommandsRange, pos):
		ch.completeCommands(ctx)
	case position.InRange(ch.Doc.ExecutorsRange, pos):
		ch.completeExecutors(ctx)
	case position.InRange(ch.Doc.OrbsRange, pos):
		if !ch.completeInInlineOrb(ctx) {
			ch.completeOrbs(ctx)
		}
	case position.InRange(ch.Doc.FunctionsRange, pos):
		ch.completeFunctionVersion(ctx)
	case position.InRange(ch.Doc.PipelineParametersRange, pos):
		ch.completeParameterDefinitions(ctx, ch.Doc.PipelineParameters, pipelineParameterTypes)
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

func (ch *CompletionHandler) GetOrbInfo(ctx context.Context, orb ast.Orb) *ast.OrbInfo {
	orbInfo, _ := ch.Doc.GetOrFetchOrbInfo(ctx, orb, ch.Cache)
	return orbInfo
}
