package documentSymbols

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"go.lsp.dev/protocol"
)

const (
	VersionSymbol       float64 = 2
	OrbSymbol           float64 = 12
	ExecutorsSymbol     float64 = 6
	CommandsSymbol      float64 = 1
	WorkflowsSymbol     float64 = 23
	PipelineParamSymbol float64 = 1
	JobSymbol           float64 = 1
	ListSymbol          float64 = 18

	StringParameterSymbol float64 = 15
	BoolParameterSymbol   float64 = 17
	EnumParameterSymbol   float64 = 10
	IntParameterSymbol    float64 = 16

	PropertySymbol float64 = 7
	TriggerSymbol  float64 = 24
	BranchSymbol   float64 = 11
	FilterSymbol   float64 = 22

	DockerSymbol float64 = 13
)

func SymbolsForDocument(document *parser.YamlDocument) []protocol.DocumentSymbol {
	symbols := []protocol.DocumentSymbol{}

	symbols = append(
		symbols,
		resolveVersionSymbol(document)...,
	)

	symbols = append(
		symbols,
		resolveOrbSymbols(document)...,
	)

	symbols = append(
		symbols,
		resolveCommandsSymbols(document)...,
	)

	symbols = append(
		symbols,
		resolveJobsSymbols(document)...,
	)

	symbols = append(
		symbols,
		resolveExecutorsSymbols(document)...,
	)

	symbols = append(
		symbols,
		resolveWorkflowsSymbols(document)...,
	)

	symbols = append(
		symbols,
		resolvePipelineParametersSymbols(document)...,
	)

	return symbols
}
