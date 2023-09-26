package definition

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func (def DefinitionStruct) searchForJobs() []protocol.Location {
	for _, job := range def.Doc.Jobs {
		if res := def.getStepDefinition(job.Steps); len(res) > 0 {
			return res
		}

		if utils.PosInRange(job.NameRange, def.Params.Position) {
			return []protocol.Location{
				{
					URI:   def.Params.TextDocument.URI,
					Range: job.Range,
				},
			}
		}

		if utils.PosInRange(job.ExecutorRange, def.Params.Position) {
			return []protocol.Location{
				{
					URI:   def.Params.TextDocument.URI,
					Range: def.getExecutorRange(job.Executor),
				},
			}
		}

		if paramDefinitions := def.searchForParamDefinition(job.Parameters); len(paramDefinitions) > 0 {
			return paramDefinitions
		}
	}

	return []protocol.Location{}
}

func (def DefinitionStruct) getStepDefinition(steps []ast.Step) []protocol.Location {
	for _, commandStep := range steps {
		switch step := commandStep.(type) {
		case ast.NamedStep:
			if utils.PosInRange(step.Range, def.Params.Position) {
				if loc, err := def.getCommandOrJobLocation(step.Name, true); err == nil {
					return loc
				}
				return []protocol.Location{}
			}

			if res := def.searchForParamValueDefinition(step.Name, step.Parameters); len(res) > 0 {
				return res
			}
		}

	}
	return []protocol.Location{}
}
