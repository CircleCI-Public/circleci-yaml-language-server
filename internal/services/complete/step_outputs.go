package complete

import (
	"bytes"
	"context"
	"regexp"
	"slices"

	"go.lsp.dev/protocol"

	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

var (
	stepIDBeingWritten     = regexp.MustCompile(`\$\{\{\s*steps\.([A-Za-z0-9_-]*)$`)
	stepOutputBeingWritten = regexp.MustCompile(`\$\{\{\s*steps\.([A-Za-z][A-Za-z0-9_-]*)\.outputs\.([A-Za-z0-9_-]*)$`)
	referenceClosing       = regexp.MustCompile(`^[A-Za-z0-9_-]*(\[\d*\])?\s*\}\}`)
)

// completeStepOutputs offers, in a reference being written to a step's
// output, the ids of the function steps before the step and, after
// `outputs.`, the outputs of the function the id names. It says whether the
// cursor is in such a reference, where a step reads outputs.
func (ch *CompletionHandler) completeStepOutputs(ctx context.Context) bool {
	line := ch.lineBeforeCursor()
	outputMatch := stepOutputBeingWritten.FindSubmatch(line)
	idMatch := stepIDBeingWritten.FindSubmatch(line)
	if outputMatch == nil && idMatch == nil {
		return false
	}

	steps, ok := ch.Doc.OutputStepsAt(ch.Params.Position)
	if !ok {
		return false
	}

	if idMatch != nil {
		for _, step := range steps {
			ch.Items = append(ch.Items, protocol.CompletionItem{
				Label:      step.ID,
				Kind:       protocol.CompletionItemKindVariable,
				Detail:     protocol.NewOptional(step.Step.Name),
				InsertText: protocol.NewOptional(step.ID + ".outputs."),
			})
		}
		return true
	}

	i := slices.IndexFunc(steps, func(step yamlparser.OutputStep) bool { return step.ID == string(outputMatch[1]) })
	if i == -1 {
		return true
	}
	_, descriptor, err := ch.Doc.LookUpFunction(ctx, steps[i].Function, ch.Cache)
	if err != nil || descriptor == nil {
		return true
	}

	closed := ch.referenceClosedAfterCursor()
	for _, output := range descriptor.DeclaredOutputs() {
		insert := output.Name
		if !closed {
			insert += " }}"
		}
		detail := output.Type
		if output.Secret {
			detail += ", secret"
		}
		item := protocol.CompletionItem{
			Label:      output.Name,
			Kind:       protocol.CompletionItemKindVariable,
			Detail:     protocol.NewOptional(detail),
			InsertText: protocol.NewOptional(insert),
		}
		if output.Description != "" {
			item.Documentation = &protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: output.Description}
		}
		ch.Items = append(ch.Items, item)
	}
	return true
}

// referenceClosedAfterCursor reports whether the reference being written is
// closed by what follows the cursor on its line, past the rest of the name and
// any index, so a completion needn't add ` }}`. A `}}` that closes something
// else, such as a template further along, doesn't count.
func (ch *CompletionHandler) referenceClosedAfterCursor() bool {
	rest := ch.Doc.Content[position.ToIndex(ch.Params.Position, ch.Doc.Content):]
	if lineEnd := bytes.IndexByte(rest, '\n'); lineEnd != -1 {
		rest = rest[:lineEnd]
	}
	return referenceClosing.Match(rest)
}
