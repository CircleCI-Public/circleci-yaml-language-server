package definition

import (
	"strings"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
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
			link, _ := def.getExecutorLink(alias.TargetRange, alias.Target)
			return []Link{link}, nil
		}
	}

	return []Link{}, nil
}

// getExecutorLink links origin to the executor named name, and reports
// whether there is one.
func (def DefinitionStruct) getExecutorLink(origin protocol.Range, name string) (Link, bool) {
	link := Link{Origin: origin, URI: def.Params.TextDocument.URI}

	if alias, ok := def.Doc.ExecutorAlias(name); ok {
		link.Range, link.NameRange = alias.Range, alias.NameRange
		return link, true
	}

	executor, ok := def.Doc.Executors[name]
	if !ok {
		orbLoc, _ := def.getOrbLocation(name, false)
		if len(orbLoc) > 0 {
			link.Range, link.NameRange = orbLoc[0].Range, orbLoc[0].NameRange
			return link, true
		}
		return link, false
	}

	link.Range, link.NameRange = executor.GetRange(), executor.GetNameRange()
	return link, true
}

// searchForExecutorArgument goes from a value given to a parameter of type
// executor, directly or among a matrix's values, to the executor it names.
func (def DefinitionStruct) searchForExecutorArgument(callName string, argument ast.ParameterValue) []Link {
	if declared, ok := def.declaredParam(callName, argument.Name); !ok || declared.GetType() != "executor" {
		return []Link{}
	}

	values := []ast.ParameterValue{argument}
	if list, ok := argument.Value.([]ast.ParameterValue); ok {
		values = list
	}
	for _, value := range values {
		name, ok := value.Value.(string)
		if !ok || !position.InRange(value.ValueRange, def.Params.Position) {
			continue
		}
		if link, ok := def.getExecutorLink(value.ValueRange, strings.Trim(name, `"'`)); ok {
			return []Link{link}
		}
	}
	return []Link{}
}
