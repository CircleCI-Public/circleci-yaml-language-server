package definition

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

type DefinitionStruct struct {
	Cache  *utils.Cache
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

	switch true {
	// Workflows
	case utils.PosInRange(def.Doc.WorkflowRange, def.Params.Position):
		return def.searchForWorkflows(), nil

	// Jobs
	case utils.PosInRange(def.Doc.JobsRange, def.Params.Position):
		return def.searchForJobs(), nil

	// Commands
	case utils.PosInRange(def.Doc.CommandsRange, def.Params.Position):
		return def.searchForCommands(), nil

	// Orbs
	case utils.PosInRange(def.Doc.OrbsRange, def.Params.Position):
		return def.getOrbDefinition()

	// Pipeline's parameters
	case utils.PosInRange(def.Doc.PipelineParametersRange, def.Params.Position):
		return def.searchForParamDefinition(def.Doc.PipelineParameters), nil

	case utils.PosInRange(def.Doc.ExecutorsRange, def.Params.Position):
		return def.getExecutorDefinition()
	}

	return nil, nil
}

func (def DefinitionStruct) GetOrbInfo(name string) (*ast.OrbInfo, error) {
	return def.Doc.GetOrbInfoFromName(name, def.Cache)
}
