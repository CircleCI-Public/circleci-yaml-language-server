package definition

import (
	"github.com/circleci/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func (def DefinitionStruct) getExecutorDefinition() ([]protocol.Location, error) {
	for _, executor := range def.Doc.Executors {
		if utils.PosInRange(executor.GetNameRange(), def.Params.Position) {
			return []protocol.Location{
				{
					URI:   def.Params.TextDocument.URI,
					Range: executor.GetRange(),
				},
			}, nil
		}
	}

	return []protocol.Location{}, nil
}

func (def DefinitionStruct) getExecutorRange(name string) protocol.Range {
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
