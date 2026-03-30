package documentSymbols

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func resolveJobGroupsSymbols(document *parser.YamlDocument) []protocol.DocumentSymbol {
	if utils.IsDefaultRange(document.JobGroupsRange) {
		return nil
	}

	jobGroupsSymbols := symbolFromRange(
		document.JobGroupsRange,
		"Job Groups",
		ListSymbol,
	)

	children := []protocol.DocumentSymbol{}

	for _, jobGroup := range document.JobGroups {
		children = append(children, singleJobGroupSymbols(jobGroup))
	}

	jobGroupsSymbols.Children = children

	return []protocol.DocumentSymbol{jobGroupsSymbols}
}

func singleJobGroupSymbols(jobGroup ast.JobGroup) protocol.DocumentSymbol {
	symbol := symbolFromRange(jobGroup.Range, jobGroup.Name, JobSymbol)

	if len(jobGroup.JobInvocations) > 0 {
		symbol.Children = append(symbol.Children, protocol.DocumentSymbol{
			Name:           "Jobs",
			Range:          jobGroup.JobsRange,
			SelectionRange: jobGroup.JobsRange,
			Children:       jobGroupJobInvocationsSymbols(jobGroup),
			Kind:           protocol.SymbolKind(ListSymbol),
		})
	}

	return symbol
}

func jobGroupJobInvocationsSymbols(jobGroup ast.JobGroup) []protocol.DocumentSymbol {
	jobs := []protocol.DocumentSymbol{}

	for _, jobInvocation := range jobGroup.JobInvocations {
		jobs = append(
			jobs,
			protocol.DocumentSymbol{
				Name:           jobInvocation.StepName,
				Range:          jobInvocation.JobInvocationRange,
				SelectionRange: jobInvocation.JobInvocationRange,
			},
		)
	}

	return jobs
}
