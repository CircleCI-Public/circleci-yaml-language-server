package definition

import (
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
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

	return def.searchForAliasTargets(def.Doc.Aliases.Commands)
}

// searchForAliasTargets goes from the target of an alias, such as `orb/c`,
// to the orb's element it names.
func (def DefinitionStruct) searchForAliasTargets(aliases map[string]ast.Alias) []protocol.Location {
	for _, alias := range aliases {
		if !position.InRange(alias.TargetRange, def.Params.Position) {
			continue
		}
		if loc, err := def.getOrbLocation(alias.Target, true); err == nil {
			return loc
		}
	}
	return []protocol.Location{}
}
