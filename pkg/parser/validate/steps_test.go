package validate

import (
	"testing"

	"go.lsp.dev/protocol"
)

func TestStepsValidation(t *testing.T) {
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
    docker:
      - image: node:latest
    steps:
      - ccc/step

workflows:
  someworkflow:
    jobs:
      - job
`,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name: "Valid usage of auto-rerun fields with proper combinations",
			YamlContent: `version: 2.1

jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Background task (valid)"
          command: "sleep 30"
          background: true
      - run:
          name: "Non-background task with max_auto_reruns only (valid)"
          command: "echo test1"
          max_auto_reruns: 3
      - run:
          name: "Non-background task with both auto-rerun fields (valid)"
          command: "echo test2"
          max_auto_reruns: 2
          auto_rerun_delay: 4m

workflows:
  test-workflow:
    jobs:
      - test-job
`,
			OnlyErrors:  true,
			Diagnostics: []protocol.Diagnostic{},
		},
	}

	CheckYamlErrors(t, testCases)
}
