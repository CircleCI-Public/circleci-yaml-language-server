package complete

import (
	"bytes"
	"regexp"
	"strings"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/pipelinevalues"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

// pipelineValueBeingWritten finds a pipeline value being written anywhere
// inside an unclosed `<<`, so in an expression too.
var pipelineValueBeingWritten = regexp.MustCompile(`<<[^>]*\bpipeline\.((?:[A-Za-z0-9_]+\.)*)[A-Za-z0-9_]*$`)

// completePipelineValues offers the next segment of the pipeline value being
// written: after `<< pipeline.git.`, that's branch, tag, revision and so on.
// The pipeline parameters are completed from the config instead.
func (ch *CompletionHandler) completePipelineValues() {
	cursor := position.ToIndex(ch.Params.Position, ch.Doc.Content)
	lineStart := position.ToIndex(protocol.Position{Line: ch.Params.Position.Line}, ch.Doc.Content)
	match := pipelineValueBeingWritten.FindSubmatch(ch.Doc.Content[lineStart:cursor])
	if match == nil || string(match[1]) == "parameters." {
		return
	}

	lineEnd := bytes.IndexByte(ch.Doc.Content[cursor:], '\n')
	if lineEnd == -1 {
		lineEnd = len(ch.Doc.Content) - cursor
	}
	closed := bytes.Contains(ch.Doc.Content[cursor:cursor+lineEnd], []byte(">>"))

	prefix := "pipeline." + string(match[1])
	offered := map[string]bool{}
	if prefix == "pipeline." && len(ch.Doc.PipelineParameters) > 0 {
		offered["parameters"] = true
		ch.Items = append(ch.Items, pipelineValueItem("parameters", pipelinevalues.Value{}, true, closed))
	}
	for _, value := range pipelinevalues.Public() {
		rest, ok := strings.CutPrefix(value.Name, prefix)
		if !ok {
			continue
		}
		segment, _, hasMore := strings.Cut(rest, ".")
		if offered[segment] {
			continue
		}
		offered[segment] = true
		ch.Items = append(ch.Items, pipelineValueItem(segment, value, hasMore, closed))
	}
}

// pipelineValueItem is the item for a segment of value's name.
func pipelineValueItem(segment string, value pipelinevalues.Value, hasMore, closed bool) protocol.CompletionItem {
	if hasMore {
		return protocol.CompletionItem{
			Label:      segment,
			Kind:       protocol.CompletionItemKindModule,
			InsertText: protocol.NewOptional(segment + "."),
		}
	}

	insert := segment
	if !closed {
		insert += " >>"
	}
	item := protocol.CompletionItem{
		Label:         segment,
		Kind:          protocol.CompletionItemKindVariable,
		Detail:        protocol.NewOptional(value.Type),
		Documentation: &protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: value.Definition},
		InsertText:    protocol.NewOptional(insert),
	}
	if value.ReplacedBy != "" {
		item.Tags = []protocol.CompletionItemTag{protocol.CompletionItemTagDeprecated}
	}
	return item
}
