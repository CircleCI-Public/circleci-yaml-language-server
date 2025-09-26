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
				}, "Only jobs with `type: approval` can be defined inline under the `workflows:` section. For `type: invalid`, define the job in the `jobs:` section instead."),
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

func TestWorkflowMaxAutoReruns(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "Valid max_auto_reruns value 1",
			YamlContent: `version: 2.1

workflows:
  test-workflow:
    max_auto_reruns: 1
    jobs:
      - hold:
          type: approval`,
			OnlyErrors:  true,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name: "Valid max_auto_reruns value 3",
			YamlContent: `version: 2.1

workflows:
  test-workflow:
    max_auto_reruns: 3
    jobs:
      - hold:
          type: approval`,
		},
		{
			Name: "Valid max_auto_reruns value 5",
			YamlContent: `version: 2.1

jobs:
	test-job:
		docker:
			- image: cimg/base:stable
		steps:
			- run: echo "test"

workflows:
	test-workflow:
		max_auto_reruns: 5
		jobs:
			- test-job`,
			OnlyErrors:  true,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name: "Invalid max_auto_reruns value 0 (below minimum)",
			YamlContent: `version: 2.1

jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"

workflows:
  test-workflow:
    max_auto_reruns: 0
    jobs:
      - test-job`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 11, Character: 21},
						End:   protocol.Position{Line: 11, Character: 22},
					},
					Severity: protocol.DiagnosticSeverityError,
					Source:   "cci-language-server",
					Message:  "Must be greater than or equal to 1",
					Data:     []protocol.CodeAction{},
				},
			},
		},
		{
			Name: "Invalid max_auto_reruns value 6 (above maximum)",
			YamlContent: `version: 2.1

jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"

workflows:
  test-workflow:
    max_auto_reruns: 6
    jobs:
      - test-job`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 11, Character: 21},
						End:   protocol.Position{Line: 11, Character: 22},
					},
					Severity: protocol.DiagnosticSeverityError,
					Source:   "cci-language-server",
					Message:  "Must be less than or equal to 5",
					Data:     []protocol.CodeAction{},
				},
			},
		},
		{
			Name: "Invalid max_auto_reruns string value",
			YamlContent: `version: 2.1

jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"

workflows:
  test-workflow:
    max_auto_reruns: "invalid"
    jobs:
      - test-job`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 11, Character: 21},
						End:   protocol.Position{Line: 11, Character: 30},
					},
					Severity: protocol.DiagnosticSeverityError,
					Source:   "cci-language-server",
					Message:  "Must be greater than or equal to 1",
					Data:     []protocol.CodeAction{},
				},
			},
		},
	}

	CheckYamlErrors(t, testCases)
}
