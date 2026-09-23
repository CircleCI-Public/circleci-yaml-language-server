package definition

import (
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func (def DefinitionStruct) searchForJobGroups() []protocol.Location {
	for _, jobGroup := range def.Doc.JobGroups {
		for _, jobInvocation := range jobGroup.JobInvocations {
			if position.InRange(jobInvocation.JobNameRange, def.Params.Position) {
				loc, err := def.getCommandOrJobLocation(jobInvocation.JobName, false)
				if err != nil {
					continue
				}
				return loc
			}

			if res := def.searchForJobInvocationFromRequires(jobInvocation.Requires, jobGroup.JobInvocations); len(res) > 0 {
				return res
			}

			if res := def.searchForParamValueDefinition(jobInvocation.JobName, jobInvocation.Parameters); len(res) > 0 {
				return res
			}
		}
	}
	return []protocol.Location{}
}
