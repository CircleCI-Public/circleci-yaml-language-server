package definition

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func (def DefinitionStruct) searchForJobInvocationFromRequires(requires []ast.Require, jobInvocations []ast.JobInvocation) []protocol.Location {
	for _, require := range requires {
		if utils.PosInRange(require.Range, def.Params.Position) {
			for _, jobInvocation := range jobInvocations {
				if jobInvocation.JobName == require.Name || jobInvocation.StepName == require.Name {
					return []protocol.Location{
						{
							URI:   def.Params.TextDocument.URI,
							Range: jobInvocation.JobInvocationRange,
						},
					}
				}
			}
		}
	}
	return []protocol.Location{}
}
