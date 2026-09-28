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
	match := pipelineValueBeingWritten.FindSubmatch(ch.lineBeforeCursor())
	if match == nil || string(match[1]) == "parameters." {
		return
	}
	closed := ch.tagClosedAfterCursor()

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

// lineBeforeCursor is the text of the cursor's line up to the cursor.
func (ch *CompletionHandler) lineBeforeCursor() []byte {
	cursor := position.ToIndex(ch.Params.Position, ch.Doc.Content)
	lineStart := position.ToIndex(protocol.Position{Line: ch.Params.Position.Line}, ch.Doc.Content)
	return ch.Doc.Content[lineStart:cursor]
}

// tagClosedAfterCursor reports whether the rest of the cursor's line closes
// the tag being written, so a completion needn't add ` >>`.
func (ch *CompletionHandler) tagClosedAfterCursor() bool {
	rest := ch.Doc.Content[position.ToIndex(ch.Params.Position, ch.Doc.Content):]
	if lineEnd := bytes.IndexByte(rest, '\n'); lineEnd != -1 {
		rest = rest[:lineEnd]
	}
	return bytes.Contains(rest, []byte(">>"))
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
