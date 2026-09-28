package definition

import (
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func (def DefinitionStruct) getExecutorDefinition() ([]protocol.Location, error) {
	for _, executor := range def.Doc.Executors {
		if position.InRange(executor.GetNameRange(), def.Params.Position) {
			return []protocol.Location{
				{
					URI:   def.Params.TextDocument.URI,
					Range: executor.GetRange(),
				},
			}, nil
		}
	}

	for _, alias := range def.Doc.Aliases.Executors {
		if position.InRange(alias.TargetRange, def.Params.Position) {
			return []protocol.Location{
				{
					URI:   def.Params.TextDocument.URI,
					Range: def.getExecutorRange(alias.Target),
				},
			}, nil
		}
	}

	return []protocol.Location{}, nil
}

func (def DefinitionStruct) getExecutorRange(name string) protocol.Range {
	if alias, ok := def.Doc.ExecutorAlias(name); ok {
		return alias.Range
	}

	executor, ok := def.Doc.Executors[name]
	if !ok {
		orbLoc, _ := def.getOrbLocation(name, false)
		if len(orbLoc) > 0 {
			return orbLoc[0].Range
		}
		return protocol.Range{}
	}

	return executor.GetRange()
}
