package documentSymbols

import (
	"fmt"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func resolveWorkflowsSymbols(document *parser.YamlDocument) []protocol.DocumentSymbol {
	if utils.IsDefaultRange(document.WorkflowRange) {
		return nil
	}

	workflowsSymbols := symbolFromRange(
		document.WorkflowRange,
		"Workflows",
		ListSymbol,
	)

	children := []protocol.DocumentSymbol{}

	for _, workflow := range document.Workflows {
		children = append(children, singleWorkflowSymbols(workflow))
	}

	workflowsSymbols.Children = children

	return []protocol.DocumentSymbol{workflowsSymbols}
}

func singleWorkflowSymbols(workflow ast.Workflow) protocol.DocumentSymbol {
	symbol := symbolFromRange(workflow.Range, workflow.Name, WorkflowsSymbol)

	if len(workflow.JobRefs) > 0 {
		symbol.Children = append(symbol.Children, protocol.DocumentSymbol{
			Name:           "Jobs",
			Range:          workflow.JobsRange,
			SelectionRange: workflow.JobsRange,
			Children:       workflowJobsSymbols(workflow),
			Kind:           protocol.SymbolKind(ListSymbol),
		})
	}

	if workflow.HasTrigger {
		symbol.Children = append(symbol.Children, protocol.DocumentSymbol{
			Name:           "Triggers",
			Range:          workflow.TriggersRange,
			SelectionRange: workflow.TriggersRange,
			Kind:           protocol.SymbolKind(TriggerSymbol),
			Children:       workflowTriggersSymbols(workflow.Triggers),
		})
	}

	return symbol
}

func workflowJobsSymbols(workflow ast.Workflow) []protocol.DocumentSymbol {
	jobs := []protocol.DocumentSymbol{}

	for _, j := range workflow.JobRefs {
		children := []protocol.DocumentSymbol{}

		if len(j.PreSteps) > 0 {
			children = append(children, protocol.DocumentSymbol{
				Name:           "Pre-Steps",
				Range:          j.PreStepsRange,
				SelectionRange: j.PreStepsRange,
				Kind:           protocol.SymbolKind(ListSymbol),
				Children:       stepsSymbols(j.PreSteps),
			})
		}

		if len(j.PostSteps) > 0 {
			children = append(children, protocol.DocumentSymbol{
				Name:           "Post-Steps",
				Range:          j.PostStepsRange,
				SelectionRange: j.PostStepsRange,
				Kind:           protocol.SymbolKind(ListSymbol),
				Children:       stepsSymbols(j.PostSteps),
			})
		}

		jobs = append(
			jobs,
			protocol.DocumentSymbol{
				Name:           j.StepName,
				Range:          j.JobRefRange,
				SelectionRange: j.JobRefRange,
				Children:       children,
			},
		)
	}

	return jobs
}

func workflowTriggersSymbols(triggers []ast.WorkflowTrigger) []protocol.DocumentSymbol {
	symbols := []protocol.DocumentSymbol{}

	for _, trigger := range triggers {
		detail := ""
		name := ""
		children := []protocol.DocumentSymbol{}

		if !utils.IsDefaultRange(trigger.Schedule.Range) {
			name = "schedule"
			detail = trigger.Schedule.Cron

			if !utils.IsDefaultRange(trigger.Schedule.Filters.Range) {
				children = append(children, filterSymbols(trigger.Schedule.Filters))
			}
		}

		if name == "" {
			continue
		}

		symbols = append(symbols, protocol.DocumentSymbol{
			Name:           name,
			Detail:         detail,
			Children:       children,
			Range:          trigger.Range,
			SelectionRange: trigger.Range,
		})
	}

	return symbols
}

func filterSymbols(filter ast.WorkflowFilters) protocol.DocumentSymbol {
	children := []protocol.DocumentSymbol{}

	if !utils.IsDefaultRange(filter.Branches.Range) {
		children = append(children, branchesFiltersSymbols(filter.Branches))
	}

	return protocol.DocumentSymbol{
		Name:           "Filter",
		Range:          filter.Range,
		SelectionRange: filter.Range,
		Children:       children,
		Kind:           protocol.SymbolKind(FilterSymbol),
	}
}

func branchesFiltersSymbols(branchesFilters ast.BranchesFilter) protocol.DocumentSymbol {
	children := []protocol.DocumentSymbol{}

	if len(branchesFilters.Ignore) > 0 {
		children = append(children, protocol.DocumentSymbol{
			Name:           "Ignore",
			Detail:         fmt.Sprintf("%d total", len(branchesFilters.Ignore)),
			Kind:           protocol.SymbolKind(BranchSymbol),
			Range:          branchesFilters.IgnoreRange,
			SelectionRange: branchesFilters.IgnoreRange,
		})
	}

	if len(branchesFilters.Only) > 0 {
		children = append(children, protocol.DocumentSymbol{
			Name:           "Only",
			Detail:         fmt.Sprintf("%d total", len(branchesFilters.Only)),
			Kind:           protocol.SymbolKind(BranchSymbol),
			Range:          branchesFilters.OnlyRange,
			SelectionRange: branchesFilters.OnlyRange,
		})
	}

	return protocol.DocumentSymbol{
		Name:           "Branches",
		Range:          branchesFilters.Range,
		SelectionRange: branchesFilters.Range,
		Children:       children,
		Kind:           protocol.SymbolKind(BranchSymbol),
	}
}
