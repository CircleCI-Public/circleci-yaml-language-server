package complete

import (
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/paramref"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	sitter "github.com/smacker/go-tree-sitter"
)

func (ch *CompletionHandler) addParametersDefinitionCompletion(parameters map[string]ast.Parameter) {
	for _, param := range parameters {
		if position.InRange(param.GetRange(), ch.Params.Position) {
			if position.InRange(param.GetTypeRange(), ch.Params.Position) {
				ch.addCompletionItem("string")
				ch.addCompletionItem("boolean")
				ch.addCompletionItem("integer")
				ch.addCompletionItem("enum")
				ch.addCompletionItem("executor")
				ch.addCompletionItem("steps")
				ch.addCompletionItem("env_var_name")
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
			}
		}
	}
}

func (ch *CompletionHandler) addParameterReferenceCompletion(node *sitter.Node) {
	if node.Type() == "string_scalar" {
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
