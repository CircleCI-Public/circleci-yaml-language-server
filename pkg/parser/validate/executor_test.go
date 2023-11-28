package validate

import (
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func TestExecutorValidation(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "Ignore workflow's jobs that are come from uncheckable orbs",
			YamlContent: `version: 2.1

parameters:
  dev-orb-version:
    type: string
    default: "dev:alpha"

orbs:
  ccc: cci-dev/ccc@<<pipeline.parameters.dev-orb-version>>

jobs:
  job:
    executor: ccc/executor
    steps:
      - run: echo "Hello"

workflows:
  someworkflow:
    jobs:
      - job
`,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name: "flag resource class error",
			YamlContent: `version: 2.1

executors:
  macos-ios-executor:
    macos:
      xcode: "15.1.0"
    resource_class: large`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 6, Character: 4},
					End:   protocol.Position{Line: 6, Character: 0x19},
				}, "Invalid resource class: \"large\""),
			},
		},
	}

	CheckYamlErrors(t, testCases)
}
