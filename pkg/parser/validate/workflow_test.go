package validate

import (
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func TestWorkflowJobRefType(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "Approval jobs",
			YamlContent: `version: 2.1

workflows:
  someworkflow:
    jobs:
      - hold:
          type: approval`,
		},
		{
			Name: "Approval jobs with quotes",
			YamlContent: `version: 2.1

workflows:
  someworkflow:
    jobs:
      - hold:
          type: "approval"`,
		},
		{
			Name: "Invalid workflow type",
			YamlContent: `version: 2.1

workflows:
  someworkflow:
    jobs:
      - hold:
          type: invalid`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 0x6, Character: 0x10},
					End:   protocol.Position{Line: 0x6, Character: 0x17},
				}, "Job Type \"invalid\" is not valid"),
			},
		},
		{
			Name: "Ignore workflow's jobs that are come from uncheckable orbs",
			YamlContent: `version: 2.1

parameters:
  dev-orb-version:
    type: string
    default: "dev:alpha"

orbs:
  ccc: cci-dev/ccc@<<pipeline.parameters.dev-orb-version>>

workflows:
  someworkflow:
    jobs:
      - ccc/job
`,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name: "Serial groups",
			YamlContent: `version: 2.1
jobs:
	- deploy:
			type: no-op

workflows:
 someworkflow:
	 jobs:
		 - deploy:
					serial-group: deploy-group`,
		},
		{
			Name: "Job override",
			YamlContent: `version: 2.1
jobs:
	- deploy:
			type: no-op

workflows:
 someworkflow:
	 jobs:
		 - deploy:
					override-with: local/deploy`,
		},
	}

	CheckYamlErrors(t, testCases)
}
