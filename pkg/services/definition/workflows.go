package definition

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func (def DefinitionStruct) searchForWorkflows() []protocol.Location {
	for _, workflow := range def.Doc.Workflows {
		for _, jobRef := range workflow.JobRefs {
			if utils.PosInRange(jobRef.JobNameRange, def.Params.Position) {
				loc, err := def.getCommandOrJobLocation(jobRef.JobName, false)
				if err != nil {
					continue
				}
				return loc
			}

			if res := def.searchForWorkflowJobsRequires(jobRef.Requires, workflow); len(res) > 0 {
				return res
			}

			if res := def.searchForParamValueDefinition(jobRef.JobName, jobRef.Parameters); len(res) > 0 {
				return res
			}
		}
	}
	return []protocol.Location{}
}

func (def DefinitionStruct) searchForWorkflowJobsRequires(requires []ast.Require, workflow ast.Workflow) []protocol.Location {
	for _, require := range requires {
		if utils.PosInRange(require.Range, def.Params.Position) {
			for _, jobRef := range workflow.JobRefs {
				if jobRef.JobName == require.Name {
					return []protocol.Location{
						{
							URI:   def.Params.TextDocument.URI,
							Range: jobRef.JobRefRange,
						},
					}
				}
			}
		}
	}
	return []protocol.Location{}
}
