package definition

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func (def DefinitionStruct) searchForWorkflows() []protocol.Location {
	for _, workflow := range def.Doc.Workflows {
		for _, jobInvocation := range workflow.JobInvocations {
			if utils.PosInRange(jobInvocation.JobNameRange, def.Params.Position) {
				loc, err := def.getCommandOrJobLocation(jobInvocation.JobName, false)
				if err != nil {
					continue
				}
				return loc
			}

			if res := def.searchForJobInvocationFromRequires(jobInvocation.Requires, workflow.JobInvocations); len(res) > 0 {
				return res
			}

			if res := def.searchForParamValueDefinition(jobInvocation.JobName, jobInvocation.Parameters); len(res) > 0 {
				return res
			}
		}
	}
	return []protocol.Location{}
}
