package definition

import (
	"context"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func (def DefinitionStruct) searchForJobInvocationFromRequires(requires []ast.Require, jobInvocations []ast.JobInvocation) []Link {
	for _, require := range requires {
		if position.InRange(require.Range, def.Params.Position) {
			for _, jobInvocation := range jobInvocations {
				if jobInvocation.JobName == require.Name || jobInvocation.StepName == require.Name {
					return []Link{
						{
							Origin:    require.Range,
							URI:       def.Params.TextDocument.URI,
							Range:     jobInvocation.JobInvocationRange,
							NameRange: jobInvocation.JobNameRange,
						},
					}
				}
			}
		}
	}
	return []Link{}
}

// searchForMatrixParamDefinition goes from a parameter of a matrix, its name
// or any of its values, to the parameter's declaration in the job, as an
// argument goes. A value of a parameter of type executor goes to the
// executor instead.
func (def DefinitionStruct) searchForMatrixParamDefinition(ctx context.Context, jobName string, matrix map[string][]ast.ParameterValue) []Link {
	for name, values := range matrix {
		for _, param := range values {
			if !position.InRange(param.Range, def.Params.Position) {
				continue
			}
			if executor := def.searchForExecutorArgument(ctx, jobName, param); len(executor) > 0 {
				return executor
			}
			if loc, err := def.getCommandOrJobParamLocation(ctx, jobName, name, true); err == nil {
				return loc
			}
			return []Link{}
		}
	}
	return []Link{}
}
