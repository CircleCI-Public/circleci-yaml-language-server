package validate

import (
	"os"
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

func TestYamlDocument_parseCheckout(t *testing.T) {
	validConfigFilePath := "./testdata/valid_checkout_method.yml"
	validConfig, err := os.ReadFile(validConfigFilePath)
	if err != nil {
		t.Fatal("Failed to read valid_checkout_method.yml")
	}

	invalidConfigFilePath := "./testdata/invalid_checkout_method.yml"
	invalidConfig, err := os.ReadFile(invalidConfigFilePath)
	if err != nil {
		t.Fatal("Failed to read invalid_checkout_method.yml")
	}

	testCases := []ValidateTestCase{
		{
			Name:        "Specifying checkout method full does not result in an error",
			YamlContent: string(validConfig),
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name:        "Specifying an invalid checkout method results in an error",
			YamlContent: string(invalidConfig),
			Diagnostics: []protocol.Diagnostic{
				{
					Severity: protocol.DiagnosticSeverityError,
					Range: protocol.Range{
						Start: protocol.Position{Line: 7, Character: 8},
						End:   protocol.Position{Line: 7, Character: 16},
					},
					Message: "Checkout method 'invalid' is invalid",
				},
			},
		},
	}
	CheckYamlErrors(t, testCases)
}
