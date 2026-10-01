package definition

import (
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func (def DefinitionStruct) getExecutorDefinition() ([]Link, error) {
	for _, executor := range def.Doc.Executors {
		if position.InRange(executor.GetNameRange(), def.Params.Position) {
			return []Link{
				{
					Origin:    executor.GetNameRange(),
					URI:       def.Params.TextDocument.URI,
					Range:     executor.GetRange(),
					NameRange: executor.GetNameRange(),
				},
			}, nil
		}
	}

	for _, alias := range def.Doc.Aliases.Executors {
		if position.InRange(alias.TargetRange, def.Params.Position) {
			return []Link{def.getExecutorLink(alias.TargetRange, alias.Target)}, nil
		}
	}

	return []Link{}, nil
}

// getExecutorLink links origin to the executor named name.
func (def DefinitionStruct) getExecutorLink(origin protocol.Range, name string) Link {
	link := Link{Origin: origin, URI: def.Params.TextDocument.URI}

	if alias, ok := def.Doc.ExecutorAlias(name); ok {
		link.Range, link.NameRange = alias.Range, alias.NameRange
		return link
	}

	executor, ok := def.Doc.Executors[name]
	if !ok {
		orbLoc, _ := def.getOrbLocation(name, false)
		if len(orbLoc) > 0 {
			link.Range, link.NameRange = orbLoc[0].Range, orbLoc[0].NameRange
		}
		return link
	}

	link.Range, link.NameRange = executor.GetRange(), executor.GetNameRange()
	return link
}
