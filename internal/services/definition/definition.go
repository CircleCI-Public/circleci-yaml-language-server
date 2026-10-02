package definition

import (
	"context"
	"log/slog"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

type DefinitionStruct struct {
	Cache  *cache.Cache
	Params protocol.DefinitionParams
	Doc    yamlparser.YamlDocument
}

// Definition is where what is at the position is defined. A link whose
// origin isn't known from what it was found by is given the node at the
// position as its origin.
func (def DefinitionStruct) Definition(ctx context.Context) ([]Link, error) {
	links, err := def.search(ctx)
	if node, _, nodeErr := position.NodeAt(def.Doc.RootNode, def.Params.Position); nodeErr == nil {
		for i := range links {
			if links[i].Origin == (protocol.Range{}) {
				links[i].Origin = protocol.Range{Start: position.Start(node), End: position.End(node)}
			}
		}
	}
	return links, err
}

func (def DefinitionStruct) search(ctx context.Context) ([]Link, error) {
	paramDefinition := def.searchParamDefinition()
	if len(paramDefinition) > 0 {
		return paramDefinition, nil
	}

	if definition := def.searchAliasDefinition(); len(definition) > 0 {
		return definition, nil
	}

	var res []Link
	var err error = nil

	switch true {
	// Job Groups
	case position.InRange(def.Doc.JobGroupsRange, def.Params.Position):
		res = def.searchForJobGroups(ctx)

	// Workflows
	case position.InRange(def.Doc.WorkflowRange, def.Params.Position):
		res = def.searchForWorkflows(ctx)

	// Jobs
	case position.InRange(def.Doc.JobsRange, def.Params.Position):
		res = def.searchForJobs(ctx)

	// Commands
	case position.InRange(def.Doc.CommandsRange, def.Params.Position):
		res = def.searchForCommands(ctx)

	// Orbs
	case position.InRange(def.Doc.OrbsRange, def.Params.Position):
		res, err = def.getOrbDefinition(ctx)

	// Pipeline's parameters
	case position.InRange(def.Doc.PipelineParametersRange, def.Params.Position):
		res, err = def.searchForParamDefinition(def.Doc.PipelineParameters), nil

	case position.InRange(def.Doc.ExecutorsRange, def.Params.Position):
		res, err = def.getExecutorDefinition(ctx)
	}

	if err != nil {
		slog.Error("error occurred during definition", "err", err)
	}
	return res, nil
}

func (def DefinitionStruct) GetOrbInfo(ctx context.Context, name string) (*ast.OrbInfo, error) {
	return def.Doc.GetOrbInfoFromName(ctx, name, def.Cache)
}
