package validate

import (
	_ "embed"
	"testing"

	"go.lsp.dev/protocol"
)

var (
	//go:embed testdata/valid_checkout_method.yml
	validCheckoutMethodYml string

	//go:embed testdata/invalid_checkout_method.yml
	invalidCheckoutMethodYml string

	//go:embed testdata/invalid_checkout_method_shallow.yml
	invalidCheckoutMethodShallowYml string

	//go:embed testdata/valid_checkout_method_shallow.yml
	validCheckoutMethodShallowYml string
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
		{
			Name: "Valid usage of max_auto_reruns with parameter expression",
			YamlContent: `version: 2.1

jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    parameters:
      retries:
        type: integer
        default: 3
    steps:
      - run:
          name: "Task with parameterized reruns"
          command: "echo test"
          max_auto_reruns: << parameters.retries >>

workflows:
  test-workflow:
    jobs:
      - test-job
`,
			OnlyErrors:  true,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name: "Valid usage of max_auto_reruns with pipeline parameter expression",
			YamlContent: `version: 2.1

parameters:
  retries:
    type: integer
    default: 2

jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with pipeline parameterized reruns"
          command: "echo test"
          max_auto_reruns: << pipeline.parameters.retries >>

workflows:
  test-workflow:
    jobs:
      - test-job
`,
			OnlyErrors:  true,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name: "Invalid usage of max_auto_reruns with out-of-range integer",
			YamlContent: `version: 2.1

jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with too many reruns"
          command: "echo test"
          max_auto_reruns: 10

workflows:
  test-workflow:
    jobs:
      - test-job
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				{
					Severity: protocol.DiagnosticSeverityError,
					Range: protocol.Range{
						Start: protocol.Position{Line: 7, Character: 8},
						End:   protocol.Position{Line: 7, Character: 11},
					},
					Message: "max_auto_reruns must be between 1 and 5",
				},
			},
		},
		{
			Name: "Invalid usage of max_auto_reruns with a non-numeric string",
			YamlContent: `version: 2.1

jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with garbage reruns"
          command: "echo test"
          max_auto_reruns: "garbage"

workflows:
  test-workflow:
    jobs:
      - test-job
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				{
					Severity: protocol.DiagnosticSeverityError,
					Range: protocol.Range{
						Start: protocol.Position{Line: 7, Character: 8},
						End:   protocol.Position{Line: 7, Character: 11},
					},
					Message: "max_auto_reruns must be between 1 and 5",
				},
			},
		},
		{
			// Guards the CheckIfOnlyParamUsed exemption: a parameter expression
			// embedded in a larger string is not deferred to compile time, so the
			// range check must still fire.
			Name: "Invalid usage of max_auto_reruns with a partial parameter expression",
			YamlContent: `version: 2.1

jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    parameters:
      retries:
        type: integer
        default: 3
    steps:
      - run:
          name: "Task with a partially interpolated rerun count"
          command: "echo test"
          max_auto_reruns: "<< parameters.retries >> 7"

workflows:
  test-workflow:
    jobs:
      - test-job
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				{
					Severity: protocol.DiagnosticSeverityError,
					Range: protocol.Range{
						Start: protocol.Position{Line: 11, Character: 8},
						End:   protocol.Position{Line: 11, Character: 11},
					},
					Message: "max_auto_reruns must be between 1 and 5",
				},
			},
		},
	}

	CheckYamlErrors(t, testCases)
}

func TestYamlDocument_parseCheckout(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name:        "Specifying checkout method full does not result in an error",
			YamlContent: validCheckoutMethodYml,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name:        "Specifying an invalid checkout method results in an error",
			YamlContent: invalidCheckoutMethodYml,
			Diagnostics: []protocol.Diagnostic{
				{
					Severity: protocol.DiagnosticSeverityError,
					Range: protocol.Range{
						Start: protocol.Position{Line: 7, Character: 8},
						End:   protocol.Position{Line: 7, Character: 16},
					},
					Message: "Checkout method 'invalid' is invalid",
				},
				{
					Severity: protocol.DiagnosticSeverityError,
					Range: protocol.Range{
						Start: protocol.Position{Line: 7, Character: 8},
						End:   protocol.Position{Line: 7, Character: 16},
					},
					Message: "Checkout depth can only be used with the shallow checkout method",
				},
			},
		},
		{
			Name:        "Specifying checkout method shallow with depth does not result in an error",
			YamlContent: validCheckoutMethodShallowYml,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name:        "Specifying checkout method shallow without depth results in an error",
			YamlContent: invalidCheckoutMethodShallowYml,
			Diagnostics: []protocol.Diagnostic{
				{
					Severity: protocol.DiagnosticSeverityError,
					Range: protocol.Range{
						Start: protocol.Position{Line: 7, Character: 8},
						End:   protocol.Position{Line: 7, Character: 16},
					},
					Message: "Checkout depth is not an integer",
				},
			},
		},
	}
	CheckYamlErrors(t, testCases)
}
