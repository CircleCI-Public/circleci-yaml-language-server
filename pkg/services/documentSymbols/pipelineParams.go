package documentSymbols

import (
	"fmt"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func resolvePipelineParametersSymbols(document *parser.YamlDocument) []protocol.DocumentSymbol {
	if utils.IsDefaultRange(document.PipelineParametersRange) {
		return nil
	}

	children := []protocol.DocumentSymbol{}

	for _, param := range document.PipelineParameters {
		children = append(children, parameterDefinitionSymbols(param))
	}

	return []protocol.DocumentSymbol{
		{
			Name:           "Pipeline Parameters",
			Kind:           1,
			Range:          document.PipelineParametersRange,
			SelectionRange: document.PipelineParametersRange,
			Children:       children,
		},
	}
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
		SelectionRange: parameter.GetRange(),
		Detail:         detail,
		Kind:           protocol.SymbolKind(PropertySymbol),
	}
}
