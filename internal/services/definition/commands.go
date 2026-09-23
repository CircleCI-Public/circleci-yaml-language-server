package definition

import (
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func (def DefinitionStruct) searchForCommands() []protocol.Location {
	for _, command := range def.Doc.Commands {
		if res := def.getStepDefinition(command.Steps); len(res) > 0 {
			return res
		}

		if position.InRange(command.NameRange, def.Params.Position) {
			return []protocol.Location{
				{
					URI:   def.Params.TextDocument.URI,
					Range: command.Range,
				},
			}
		}

		if paramDefinitions := def.searchForParamDefinition(command.Parameters); len(paramDefinitions) > 0 {
			return paramDefinitions
		}
	}

	return []protocol.Location{}
}
