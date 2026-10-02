package definition

import (
	"context"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func (def DefinitionStruct) searchForJobGroups(ctx context.Context) []Link {
	for _, jobGroup := range def.Doc.JobGroups {
		for _, jobInvocation := range jobGroup.JobInvocations {
			if position.InRange(jobInvocation.JobNameRange, def.Params.Position) {
				loc, err := def.getCommandOrJobLocation(ctx, jobInvocation.JobName, false)
				if err != nil {
					continue
				}
				return from(jobInvocation.JobNameRange, loc)
			}

			if res := def.searchForJobInvocationFromRequires(jobInvocation.Requires, jobGroup.JobInvocations); len(res) > 0 {
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
