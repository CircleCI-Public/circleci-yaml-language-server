package documentSymbols

import (
	"fmt"

	"go.lsp.dev/protocol"

	ast2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func resolveJobsSymbols(document *parser.YamlDocument) []protocol.DocumentSymbol {
	if position.IsDefaultRange(document.JobsRange) {
		return nil
	}

	jobsSymbols := sectionSymbol(
		document,
		"jobs",
		document.JobsRange,
		"Jobs",
	)

	children := []protocol.DocumentSymbol{}

	for _, job := range document.Jobs {
		children = append(children, singleJobSymbols(job))
	}

	jobsSymbols.Children = children

	return []protocol.DocumentSymbol{jobsSymbols}
}

func singleJobSymbols(job ast2.Job) protocol.DocumentSymbol {
	jobSymbol := symbolFromRange(job.Range, job.Name, JobSymbol)
	jobSymbol.SelectionRange = selectionRange(job.Range, job.NameRange)

	if !position.IsDefaultRange(job.ParametersRange) {
		jobSymbol.Children = append(jobSymbol.Children, protocol.DocumentSymbol{
			Name:           "Parameters",
			Kind:           ListSymbol,
			Range:          job.ParametersRange,
			SelectionRange: job.ParametersRange,
			Children:       parametersSymbols(job.Parameters),
		})
	}

	if !position.IsDefaultRange(job.StepsRange) {
		jobSymbol.Children = append(jobSymbol.Children, protocol.DocumentSymbol{
			Name:           "Steps",
			Range:          job.StepsRange,
			SelectionRange: job.StepsRange,
			Children:       stepsSymbols(job.Steps),
			Kind:           ListSymbol,
			Detail:         unlessZero(fmt.Sprintf("%d total", len(job.Steps))),
		})
	}

	if !position.IsDefaultRange(job.ExecutorRange) && job.Executor != "" {
		jobSymbol.Children = append(jobSymbol.Children, protocol.DocumentSymbol{
			Name:           fmt.Sprintf("Executor: %s", job.Executor),
			Range:          job.ExecutorRange,
			SelectionRange: job.ExecutorRange,
			Kind:           ExecutorSymbol,
		})
	}

	if !position.IsDefaultRange(job.DockerRange) {
		jobSymbol.Children = append(jobSymbol.Children, dockerExecutorSymbols(job.Docker))
	}

	if !position.IsDefaultRange(job.EnvironmentRange) {
		jobSymbol.Children = append(
			jobSymbol.Children,
			envsSymbols(
				ast2.Environment{
					Range:     job.EnvironmentRange,
					Variables: job.EnvironmentVariables,
				},
			),
		)
	}

	if !position.IsDefaultRange(job.MacOSRange) {
		jobSymbol.Children = append(jobSymbol.Children, protocol.DocumentSymbol{
			Name:           "MacOS",
			Range:          job.MacOSRange,
			SelectionRange: job.MacOSRange,
			Kind:           ExecutorSymbol,
			Children: []protocol.DocumentSymbol{
				{
					Name:           "xcode",
					Kind:           PropertySymbol,
					Detail:         unlessZero(job.MacOS.Xcode),
					Range:          job.MacOS.Range,
					SelectionRange: job.MacOS.Range,
				},
			},
		})
	}

	return jobSymbol
}

func parametersSymbols(parameters map[string]ast2.Parameter) []protocol.DocumentSymbol {
	symbols := []protocol.DocumentSymbol{}

	for _, param := range parameters {
		symbols = append(symbols, parameterDefinitionSymbols(param))
	}

	return symbols
}

func stepsSymbols(steps []ast2.Step) []protocol.DocumentSymbol {
	symbols := []protocol.DocumentSymbol{}

	for _, step := range steps {
		symbols = append(symbols, protocol.DocumentSymbol{
			Name:           step.GetName(),
			Range:          step.GetRange(),
			SelectionRange: step.GetRange(),
			Kind:           InvocationSymbol,
		})
	}

	return symbols
}
