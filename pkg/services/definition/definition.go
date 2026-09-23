package definition

import (
	"log/slog"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"go.lsp.dev/protocol"
)

type DefinitionStruct struct {
	Cache  *cache.Cache
	Params protocol.DefinitionParams
	Doc    yamlparser.YamlDocument
}

func (def DefinitionStruct) Definition() ([]protocol.Location, error) {
	paramDefinition := def.searchParamDefinition()
	if len(paramDefinition) > 0 {
		return paramDefinition, nil
	}

	if definition := def.searchAliasDefinition(); len(definition) > 0 {
		return definition, nil
	}

	var res []protocol.Location
	var err error = nil

	switch true {
	// Job Groups
	case position.InRange(def.Doc.JobGroupsRange, def.Params.Position):
		res = def.searchForJobGroups()

	// Workflows
	case position.InRange(def.Doc.WorkflowRange, def.Params.Position):
		res = def.searchForWorkflows()

	// Jobs
	case position.InRange(def.Doc.JobsRange, def.Params.Position):
		res = def.searchForJobs()

	// Commands
	case position.InRange(def.Doc.CommandsRange, def.Params.Position):
		res = def.searchForCommands()

	// Orbs
	case position.InRange(def.Doc.OrbsRange, def.Params.Position):
		res, err = def.getOrbDefinition()

	// Pipeline's parameters
	case position.InRange(def.Doc.PipelineParametersRange, def.Params.Position):
		res, err = def.searchForParamDefinition(def.Doc.PipelineParameters), nil

	case position.InRange(def.Doc.ExecutorsRange, def.Params.Position):
		res, err = def.getExecutorDefinition()
	}

	if err != nil {
		slog.Error("error occurred during definition", "err", err)
	}
	return res, nil
}

func (def DefinitionStruct) GetOrbInfo(name string) (*ast.OrbInfo, error) {
	return def.Doc.GetOrbInfoFromName(name, def.Cache)
}
