package documentSymbols

import (
	"fmt"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func resolveJobsSymbols(document *parser.YamlDocument) []protocol.DocumentSymbol {
	if utils.IsDefaultRange(document.JobsRange) {
		return nil
	}

	jobsSymbols := symbolFromRange(
		document.JobsRange,
		"Jobs",
		ListSymbol,
	)

	children := []protocol.DocumentSymbol{}

	for _, job := range document.Jobs {
		children = append(children, singleJobSymbols(job))
	}

	jobsSymbols.Children = children

	return []protocol.DocumentSymbol{jobsSymbols}
}

func singleJobSymbols(job ast.Job) protocol.DocumentSymbol {
	jobSymbol := symbolFromRange(job.Range, job.Name, JobSymbol)

	if !utils.IsDefaultRange(job.ParametersRange) {
		jobSymbol.Children = append(jobSymbol.Children, protocol.DocumentSymbol{
			Name:           "Parameters",
			Range:          job.ParametersRange,
			SelectionRange: job.ParametersRange,
			Children:       parametersSymbols(job.Parameters),
		})
	}

	if !utils.IsDefaultRange(job.StepsRange) {
		jobSymbol.Children = append(jobSymbol.Children, protocol.DocumentSymbol{
			Name:           "Steps",
			Range:          job.StepsRange,
			SelectionRange: job.StepsRange,
			Children:       stepsSymbols(job.Steps),
			Kind:           protocol.SymbolKind(ListSymbol),
			Detail:         fmt.Sprintf("%d total", len(job.Steps)),
		})
	}

	if !utils.IsDefaultRange(job.ExecutorRange) && job.Executor != "" {
		jobSymbol.Children = append(jobSymbol.Children, protocol.DocumentSymbol{
			Name:           fmt.Sprintf("Executor: %s", job.Executor),
			Range:          job.ExecutorRange,
			SelectionRange: job.ExecutorRange,
			Kind:           protocol.SymbolKind(ExecutorsSymbol),
		})
	}

	if !utils.IsDefaultRange(job.DockerRange) {
		jobSymbol.Children = append(jobSymbol.Children, dockerExecutorSymbols(job.Docker))
	}

	if !utils.IsDefaultRange(job.EnvironmentRange) {
		keys := []string{}

		for k := range job.Environment {
			keys = append(keys, k)
		}

		jobSymbol.Children = append(
			jobSymbol.Children,
			envsSymbols(
				ast.Environment{
					Range: job.EnvironmentRange,
					Keys:  keys,
				},
			),
		)
	}

	return jobSymbol
}

func parametersSymbols(parameters map[string]ast.Parameter) []protocol.DocumentSymbol {
	symbols := []protocol.DocumentSymbol{}

	for _, param := range parameters {
		symbols = append(symbols, parameterDefinitionSymbols(param))
	}

	return symbols
}

func stepsSymbols(steps []ast.Step) []protocol.DocumentSymbol {
	symbols := []protocol.DocumentSymbol{}

	for _, step := range steps {
		symbols = append(symbols, protocol.DocumentSymbol{
			Name:           step.GetName(),
			Range:          step.GetRange(),
			SelectionRange: step.GetRange(),
			Kind:           protocol.SymbolKind(JobSymbol),
		})
	}

	return symbols
}

func jobDockerExecutorSymbol(job ast.Job) protocol.DocumentSymbol {
	symbol := protocol.DocumentSymbol{
		Name:           "Docker",
		Range:          job.DockerRange,
		SelectionRange: job.DockerRange,
		Kind:           protocol.SymbolKind(DockerSymbol),
	}

	for _, img := range job.Docker.Image {
		name := img.Name

		if name == "" {
			name = img.Image.FullPath
		}

		if name == "" {
			continue
		}

		symbol.Children = append(symbol.Children, protocol.DocumentSymbol{
			Name:           name,
			Range:          img.ImageRange,
			SelectionRange: img.ImageRange,
			Kind:           8,
		})
	}

	return symbol
}
