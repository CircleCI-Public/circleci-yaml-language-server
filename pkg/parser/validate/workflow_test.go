package validate

import (
	"testing"

	"go.lsp.dev/protocol"
)

func TestWorkflowMaxAutoReruns(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "Valid max_auto_reruns value 1",
			YamlContent: `version: 2.1

jobs:
  hold:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "hold"

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

jobs:
  hold:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "hold"

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
