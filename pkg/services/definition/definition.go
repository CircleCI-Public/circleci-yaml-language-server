package definition

import (
	yamlparser "github.com/circleci/circleci-yaml-language-server/pkg/parser"
	"github.com/circleci/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

type DefinitionStruct struct {
	Cache  utils.Cache
	Params protocol.DefinitionParams
	Doc    yamlparser.YamlDocument
}

func (def DefinitionStruct) Definition(yamlDocument yamlparser.YamlDocument) ([]protocol.Location, error) {
	paramDefinition := def.searchParamDefinition()
	if len(paramDefinition) > 0 {
		return paramDefinition, nil
	}

	if definition := def.searchAliasDefinition(yamlDocument); len(definition) > 0 {
		return definition, nil
	}

	switch true {
	// Workflows
	case utils.PosInRange(yamlDocument.WorkflowRange, def.Params.Position):
		return def.searchForWorkflows(), nil

	// Jobs
	case utils.PosInRange(yamlDocument.JobsRange, def.Params.Position):
		return def.searchForJobs(), nil

	// Commands
	case utils.PosInRange(yamlDocument.CommandsRange, def.Params.Position):
		return def.searchForCommands(), nil

	// Orbs
	case utils.PosInRange(yamlDocument.OrbsRange, def.Params.Position):
		return def.getOrbDefinition()

	// Pipeline's parameters
	case utils.PosInRange(yamlDocument.PipelinesParametersRange, def.Params.Position):
		return def.searchForParamDefinition(yamlDocument.PipelinesParameters), nil

	case utils.PosInRange(yamlDocument.ExecutorsRange, def.Params.Position):
		return def.getExecutorDefinition()
	}

	return nil, nil
}
