package complete

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/paramref"
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

func (ch *CompletionHandler) addParameterReferenceCompletion(node *sitter.Node) {
	if node.Kind() == "string_scalar" {
		isParamBeingWritten, isPipelineParam := paramref.IsPartiallyReferenced(ch.Doc.GetNodeText(node))
		if isParamBeingWritten {
			if isPipelineParam {
				ch.addPipelineParametersReferenceCompletion()
			} else {
				ch.addParametersReferenceCompletion()
			}
		}
	}
}

func (ch *CompletionHandler) addPipelineParametersReferenceCompletion() {
	shouldAddClosingBrackets := ch.shouldAddParamsClosingBrackets()
	for _, param := range ch.Doc.PipelineParameters {
		if shouldAddClosingBrackets {
			ch.addCompletionItemFieldWithCustomText(param.GetName(), "", " >>", "", "")
		} else {
			ch.addCompletionItem(param.GetName())
		}
	}
}

func (ch *CompletionHandler) addParametersReferenceCompletion() {
	shouldAddClosingBrackets := ch.shouldAddParamsClosingBrackets()
	for _, param := range ch.Doc.GetParamsWithPosition(ch.Params.Position) {
		if shouldAddClosingBrackets {
			ch.addCompletionItemFieldWithCustomText(param.GetName(), "", " >>", "", "")
		} else {
			ch.addCompletionItem(param.GetName())
		}
	}
}

func (ch *CompletionHandler) shouldAddParamsClosingBrackets() bool {
	idx := position.ToIndex(ch.Params.Position, ch.Doc.Content)

	if strings.HasPrefix(string(ch.Doc.Content[idx:]), " >>") ||
		strings.HasPrefix(string(ch.Doc.Content[idx:]), ">>") {
		return false
	}

	return true
}
