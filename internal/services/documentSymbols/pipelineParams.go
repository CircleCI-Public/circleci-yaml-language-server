package documentSymbols

import (
	"fmt"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func resolvePipelineParametersSymbols(document *parser.YamlDocument) []protocol.DocumentSymbol {
	if position.IsDefaultRange(document.PipelineParametersRange) {
		return nil
	}

	children := []protocol.DocumentSymbol{}

	for _, param := range document.PipelineParameters {
		children = append(children, parameterDefinitionSymbols(param))
	}

	symbol := sectionSymbol(document, "parameters", document.PipelineParametersRange, "Pipeline Parameters")
	symbol.Children = children

	return []protocol.DocumentSymbol{symbol}
}

func parameterDefinitionSymbols(parameter ast.Parameter) protocol.DocumentSymbol {
	paramType := parameter.GetType()

	detail := fmt.Sprintf(
		"[%s]",
		paramType,
	)

	if parameter.GetDescription() != "" {
		detail = fmt.Sprintf("%s - %s", detail, parameter.GetDescription())
	}

	return protocol.DocumentSymbol{
		Name:           parameter.GetName(),
		Range:          parameter.GetRange(),
		SelectionRange: selectionRange(parameter.GetRange(), parameter.GetNameRange()),
		Detail:         unlessZero(detail),
		Kind:           PropertySymbol,
	}
}
