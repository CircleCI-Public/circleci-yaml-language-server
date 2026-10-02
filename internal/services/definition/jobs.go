package definition

import (
	"context"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func (def DefinitionStruct) searchForJobs(ctx context.Context) []Link {
	for _, job := range def.Doc.Jobs {
		if res := def.getStepDefinition(ctx, job.Steps); len(res) > 0 {
			return res
		}

		if position.InRange(job.NameRange, def.Params.Position) {
			return []Link{
				{
					Origin:    job.NameRange,
					URI:       def.Params.TextDocument.URI,
					Range:     job.Range,
					NameRange: job.NameRange,
				},
			}
		}

		if position.InRange(job.ExecutorRange, def.Params.Position) {
			link, _ := def.getExecutorLink(ctx, job.ExecutorRange, job.Executor)
			return []Link{link}
		}

		if paramDefinitions := def.searchForParamDefinition(job.Parameters); len(paramDefinitions) > 0 {
			return paramDefinitions
		}
	}

	return def.searchForAliasTargets(ctx, def.Doc.Aliases.Jobs)
}

func (def DefinitionStruct) getStepDefinition(ctx context.Context, steps []ast.Step) []Link {
	for _, commandStep := range steps {
		switch step := commandStep.(type) {
		case ast.NamedStep:
			if position.InRange(step.Range, def.Params.Position) {
				if loc, err := def.getCommandOrJobLocation(ctx, step.Name, true); err == nil {
					return loc
				}
				return []Link{}
			}

			if res := def.searchForParamValueDefinition(ctx, step.Name, step.Parameters); len(res) > 0 {
				return res
			}
		}

	}
	return []Link{}
}
