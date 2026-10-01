package definition

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func (def DefinitionStruct) searchForCommands() []Link {
	for _, command := range def.Doc.Commands {
		if res := def.getStepDefinition(command.Steps); len(res) > 0 {
			return res
		}

		if position.InRange(command.NameRange, def.Params.Position) {
			return []Link{
				{
					Origin:    command.NameRange,
					URI:       def.Params.TextDocument.URI,
					Range:     command.Range,
					NameRange: command.NameRange,
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
func (def DefinitionStruct) searchForAliasTargets(aliases map[string]ast.Alias) []Link {
	for _, alias := range aliases {
		if !position.InRange(alias.TargetRange, def.Params.Position) {
			continue
		}
		if loc, err := def.getOrbLocation(alias.Target, true); err == nil {
			return from(alias.TargetRange, loc)
		}
	}
	return []Link{}
}
