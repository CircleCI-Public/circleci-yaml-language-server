package complete

import (
	"bytes"
	"regexp"
	"slices"
	"strings"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/pipelinevalues"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

// pipelineValueBeingWritten finds a pipeline value being written at the end
// of an expression.
var pipelineValueBeingWritten = regexp.MustCompile(`(?:^|[^\w.])pipeline\.((?:[A-Za-z0-9_]+\.)*)[A-Za-z0-9_]*$`)

// wordBeingWritten finds a name being started at the end of an expression.
var wordBeingWritten = regexp.MustCompile(`(?:^|[^\w.])(\w+)$`)

// completePipelineValues offers the next segment of the pipeline value being
// written: after `<< pipeline.git.`, that's branch, tag, revision and so on.
// The pipeline parameters are completed from the config instead. In a bare
// expression, `pipeline` itself is offered too.
func (ch *CompletionHandler) completePipelineValues() {
	expression, inTag, ok := ch.expressionBeforeCursor()
	if !ok {
		return
	}
	match := pipelineValueBeingWritten.FindSubmatch(expression)
	if match == nil {
		if word := wordBeingWritten.FindSubmatch(expression); !inTag && word != nil &&
			strings.HasPrefix("pipeline", string(word[1])) {
			ch.Items = append(ch.Items, pipelineValueItem("pipeline", pipelinevalues.Value{}, true, true))
		}
		return
	}
	if string(match[1]) == "parameters." {
		return
	}
	closed := !inTag || ch.tagClosedAfterCursor()

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

// expressionBeforeCursor is the expression the cursor is in, up to the
// cursor. That's the rest of an unclosed `<<` on the cursor's line, or else,
// if the cursor is in a workflow's condition or filter written as a bare
// expression, the line. inTag is whether it's in a `<<`, where a completion
// closes the tag and a job's parameters can be used.
func (ch *CompletionHandler) expressionBeforeCursor() (expression []byte, inTag bool, ok bool) {
	line := ch.lineBeforeCursor()
	if open := bytes.LastIndex(line, []byte("<<")); open != -1 {
		expression = line[open+len("<<"):]
		return expression, true, !bytes.Contains(expression, []byte(">>"))
	}
	return line, false, ch.inBareExpression()
}

// inBareExpression reports whether the cursor is in a workflow's condition or
// a job's filter written as an expression. A step's condition isn't one.
func (ch *CompletionHandler) inBareExpression() bool {
	inOne := func(values []ast.TextAndRange) bool {
		return slices.ContainsFunc(values, func(value ast.TextAndRange) bool {
			return !strings.Contains(value.Text, "<<") && position.InRange(value.Range, ch.Params.Position) &&
				!slices.Contains(ch.Doc.StepConditions, value)
		})
	}
	return inOne(ch.Doc.Conditions) || inOne(ch.Doc.FilterExpressions)
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
