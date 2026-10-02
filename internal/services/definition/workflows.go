package definition

import (
	"context"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func (def DefinitionStruct) searchForWorkflows(ctx context.Context) []Link {
	for _, workflow := range def.Doc.Workflows {
		for _, jobInvocation := range workflow.JobInvocations {
			if position.InRange(jobInvocation.JobNameRange, def.Params.Position) {
				loc, err := def.getCommandOrJobLocation(ctx, jobInvocation.JobName, false)
				if err != nil {
					continue
				}
				return from(jobInvocation.JobNameRange, loc)
			}

			if res := def.searchForJobInvocationFromRequires(jobInvocation.Requires, workflow.JobInvocations); len(res) > 0 {
				return res
			}

			if res := def.searchForParamValueDefinition(ctx, jobInvocation.JobName, jobInvocation.Parameters); len(res) > 0 {
				return res
			}

			if res := def.searchForMatrixParamDefinition(ctx, jobInvocation.JobName, jobInvocation.MatrixParams); len(res) > 0 {
				return res
			}
		}
	}
	return []Link{}
}
