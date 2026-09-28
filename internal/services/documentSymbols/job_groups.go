package documentSymbols

import (
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func resolveJobGroupsSymbols(document *parser.YamlDocument) []protocol.DocumentSymbol {
	if position.IsDefaultRange(document.JobGroupsRange) {
		return nil
	}

	jobGroupsSymbols := sectionSymbol(
		document,
		"job-groups",
		document.JobGroupsRange,
		"Job Groups",
	)

	children := []protocol.DocumentSymbol{}

	for _, jobGroup := range document.JobGroups {
		children = append(children, singleJobGroupSymbols(jobGroup))
	}

	jobGroupsSymbols.Children = children

	return []protocol.DocumentSymbol{jobGroupsSymbols}
}

func singleJobGroupSymbols(jobGroup ast.JobGroup) protocol.DocumentSymbol {
	symbol := symbolFromRange(jobGroup.Range, jobGroup.Name, WorkflowSymbol)
	symbol.SelectionRange = selectionRange(jobGroup.Range, jobGroup.NameRange)

	if len(jobGroup.JobInvocations) > 0 {
		symbol.Children = append(symbol.Children, protocol.DocumentSymbol{
			Name:           "Jobs",
			Range:          jobGroup.JobsRange,
			SelectionRange: jobGroup.JobsRange,
			Children:       jobGroupJobInvocationsSymbols(jobGroup),
			Kind:           ListSymbol,
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
				Kind:           InvocationSymbol,
				Range:          jobInvocation.JobInvocationRange,
				SelectionRange: selectionRange(jobInvocation.JobInvocationRange, jobInvocation.StepNameRange),
			},
		)
	}

	return jobs
}
