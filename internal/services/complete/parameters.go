package complete

import (
	"maps"
	"regexp"
	"slices"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

var parameterTypes = []string{"string", "boolean", "integer", "enum", "executor", "steps", "env_var_name"}

// pipelineParameterTypes leaves out executor, steps and env_var_name, which
// only mean something inside a job, a command or an executor.
var pipelineParameterTypes = []string{"string", "boolean", "integer", "enum"}

func (ch *CompletionHandler) addParametersDefinitionCompletion(parameters map[string]ast.Parameter) {
	ch.completeParameterDefinitions(parameters, parameterTypes)
}

// completeParameterDefinitions completes in the definition of one of the
// parameters, whose type is one of types.
func (ch *CompletionHandler) completeParameterDefinitions(parameters map[string]ast.Parameter, types []string) {
	for _, param := range parameters {
		if position.InRange(param.GetRange(), ch.Params.Position) {
			if position.InRange(param.GetTypeRange(), ch.Params.Position) {
				for _, paramType := range types {
					ch.addCompletionItem(paramType)
				}
				return
			}
			if param.GetType() == "enum" && position.InRange(param.GetDefaultRange(), ch.Params.Position) {
				param := param.(ast.EnumParameter)
				for _, value := range param.Enum {
					ch.addCompletionItem(value)
				}
				return
			}

			if param.GetType() == "boolean" {
				if position.InRange(param.GetDefaultRange(), ch.Params.Position) {
					ch.addCompletionItem("true")
					ch.addCompletionItem("false")
					return
				}
			}

			if param.GetType() == "executor" {
				if position.InRange(param.GetDefaultRange(), ch.Params.Position) {
					ch.addExecutorsCompletion()
					return
				}
			}

			if param.GetTypeRange().Start.Line == 0 && param.GetTypeRange().Start.Character == 0 {
				ch.addCompletionItemField("type")
			} else {
				// Only suggest other fields if the type is defined
				if param.GetDefaultRange().Start.Line == 0 && param.GetDefaultRange().Start.Character == 0 {
					ch.addCompletionItemField("default")
				}
				if param.GetDescription() == "" {
					ch.addCompletionItemField("description")
				}
				if enum, ok := param.(ast.EnumParameter); ok && len(enum.Enum) == 0 {
					ch.addCompletionItemField("enum")
				}
			}
		}
	}
}

// parameterBeingWritten finds a parameter being written at the end of an
// expression. The first group is `pipeline.` for a pipeline parameter.
var parameterBeingWritten = regexp.MustCompile(`(?:^|[^\w.])(pipeline\.)?parameters\.[\w-]*$`)

// completeParameterReferences offers the parameters after `parameters.`,
// those of the job, command or executor the cursor is in, or after
// `pipeline.parameters.`, the pipeline's. A bare expression only has the
// pipeline's.
func (ch *CompletionHandler) completeParameterReferences() {
	expression, inTag, ok := ch.expressionBeforeCursor()
	if !ok {
		return
	}
	match := parameterBeingWritten.FindSubmatch(expression)
	if match == nil || !inTag && len(match[1]) == 0 {
		return
	}
	parameters := ch.Doc.GetParamsWithPosition(ch.Params.Position)
	if len(match[1]) > 0 {
		parameters = ch.Doc.PipelineParameters
	}

	closed := !inTag || ch.tagClosedAfterCursor()
	for _, name := range slices.Sorted(maps.Keys(parameters)) {
		param := parameters[name]
		insert := param.GetName()
		if !closed {
			insert += " >>"
		}
		item := protocol.CompletionItem{
			Label:      param.GetName(),
			Kind:       protocol.CompletionItemKindVariable,
			Detail:     protocol.NewOptional(param.GetType()),
			InsertText: protocol.NewOptional(insert),
		}
		if param.GetDescription() != "" {
			item.Documentation = &protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: param.GetDescription()}
		}
		ch.Items = append(ch.Items, item)
	}
}
