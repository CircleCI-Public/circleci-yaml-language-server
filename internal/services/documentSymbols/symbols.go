package documentSymbols

import (
	"cmp"
	"slices"
	"strings"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
)

const (
	SectionSymbol    = protocol.SymbolKindNamespace
	PropertySymbol   = protocol.SymbolKindProperty
	OrbSymbol        = protocol.SymbolKindModule
	CommandSymbol    = protocol.SymbolKindFunction
	JobSymbol        = protocol.SymbolKindFunction
	ExecutorSymbol   = protocol.SymbolKindClass
	WorkflowSymbol   = protocol.SymbolKindStruct
	InvocationSymbol = protocol.SymbolKindMethod
	ListSymbol       = protocol.SymbolKindArray
	VariableSymbol   = protocol.SymbolKindVariable
	DockerSymbol     = protocol.SymbolKindVariable
	ImageSymbol      = protocol.SymbolKindField
	TriggerSymbol    = protocol.SymbolKindEvent
	FilterSymbol     = protocol.SymbolKindObject
)

func SymbolsForDocument(document *parser.YamlDocument) []protocol.DocumentSymbol {
	symbols := []protocol.DocumentSymbol{}

	symbols = append(
		symbols,
		resolveVersionSymbol(document)...,
	)

	symbols = append(
		symbols,
		resolveSetupSymbol(document)...,
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
		resolveJobGroupsSymbols(document)...,
	)

	symbols = append(
		symbols,
		resolvePipelineParametersSymbols(document)...,
	)

	symbols = withoutBlankNames(symbols)
	inSourceOrder(symbols)

	return symbols
}

// inSourceOrder sorts each level of the tree by where its symbols start, as
// most come from maps. Symbols sharing a range, such as the keys of an
// environment, are sorted by name.
func inSourceOrder(symbols []protocol.DocumentSymbol) {
	slices.SortFunc(symbols, func(a, b protocol.DocumentSymbol) int {
		return cmp.Or(
			cmp.Compare(a.Range.Start.Line, b.Range.Start.Line),
			cmp.Compare(a.Range.Start.Character, b.Range.Start.Character),
			strings.Compare(a.Name, b.Name),
		)
	})

	for i := range symbols {
		inSourceOrder(symbols[i].Children)
	}
}

// withoutBlankNames drops symbols whose name is empty or only whitespace, such
// as a half-typed `- ` step. The LSP spec forbids them, and VS Code rejects
// the whole response if it finds one.
func withoutBlankNames(symbols []protocol.DocumentSymbol) []protocol.DocumentSymbol {
	kept := make([]protocol.DocumentSymbol, 0, len(symbols))

	for _, symbol := range symbols {
		if strings.TrimSpace(symbol.Name) == "" {
			continue
		}

		if symbol.Children != nil {
			symbol.Children = withoutBlankNames(symbol.Children)
		}

		kept = append(kept, symbol)
	}

	return kept
}
