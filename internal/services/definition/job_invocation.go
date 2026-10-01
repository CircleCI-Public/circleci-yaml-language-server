package definition

import (
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
