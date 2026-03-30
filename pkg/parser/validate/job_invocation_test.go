package validate

import (
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func TestJobInvocationType(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "Approval jobs defined in a job invocation don't need to be defined under jobs section",
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
  deploy:
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
  deploy:
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

func TestTerminalJobStatusesHint(t *testing.T) {
	testCases := []struct {
		name            string
		yamlContent     string
		expectHint      bool
		expectedNewText string
		expectedRange   protocol.Range
	}{
		{
			name: "All terminal statuses present - should show hint",
			yamlContent: `version: 2.1

jobs:
  job1:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"
  job2:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"

workflows:
  test-workflow:
    jobs:
      - job1
      - job2:
          requires:
            - job1: [success, failed, canceled, not_run, unauthorized]`,
			expectHint:      true,
			expectedNewText: "terminal",
			expectedRange: protocol.Range{
				Start: protocol.Position{Line: 20, Character: 20},
				End:   protocol.Position{Line: 20, Character: 70},
			},
		},
		{
			name: "All terminal statuses as YAML list - should show hint",
			yamlContent: `version: 2.1

jobs:
  job1:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"
  job2:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"

workflows:
  test-workflow:
    jobs:
      - job1
      - job2:
          requires:
            - job1:
              - success
              - failed
              - canceled
              - not_run
              - unauthorized`,
			expectHint:      true,
			expectedNewText: " terminal",
			expectedRange: protocol.Range{
				Start: protocol.Position{Line: 20, Character: 19},
				End:   protocol.Position{Line: 25, Character: 28},
			},
		},
		{
			name: "All terminal statuses in different order - should show hint",
			yamlContent: `version: 2.1

jobs:
  job1:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"
  job2:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"

workflows:
  test-workflow:
    jobs:
      - job1
      - job2:
          requires:
            - job1: [not_run, canceled, failed, success, unauthorized]`,
			expectHint:      true,
			expectedNewText: "terminal",
			expectedRange: protocol.Range{
				Start: protocol.Position{Line: 20, Character: 20},
				End:   protocol.Position{Line: 20, Character: 70},
			},
		},
		{
			name: "Only some terminal statuses - no hint",
			yamlContent: `version: 2.1

jobs:
  job1:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"
  job2:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"

workflows:
  test-workflow:
    jobs:
      - job1
      - job2:
          requires:
            - job1: [success, failed]`,
			expectHint: false,
		},
		{
			name: "Mix of terminal and non-terminal statuses - no hint",
			yamlContent: `version: 2.1

jobs:
  job1:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"
  job2:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"

workflows:
  test-workflow:
    jobs:
      - job1
      - job2:
          requires:
            - job1: [success, failed, canceled, not_run, unauthorized, unknown]`,
			expectHint: false,
		},
		{
			name: "All terminal statuses with anchor - should show hint",
			yamlContent: `version: 2.1

jobs:
  job1:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"
  job2:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"

workflows:
  test-workflow:
    jobs:
      - job1
      - job2:
          requires:
            - job1: &terminal-statuses [success, failed, canceled, not_run, unauthorized]`,
			expectHint:      true,
			expectedNewText: "terminal",
			expectedRange: protocol.Range{
				Start: protocol.Position{Line: 20, Character: 39},
				End:   protocol.Position{Line: 20, Character: 89},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			val := CreateValidateFromYAML(tt.yamlContent)
			val.Validate()

			diags := *val.Diagnostics

			// Filter for only Hint severity diagnostics
			hintDiags := []protocol.Diagnostic{}
			for _, d := range diags {
				if d.Severity == protocol.DiagnosticSeverityHint {
					hintDiags = append(hintDiags, d)
				}
			}

			if !tt.expectHint {
				if len(hintDiags) != 0 {
					t.Errorf("Expected no hint diagnostics, got %d", len(hintDiags))
				}
				return
			}

			if len(hintDiags) != 1 {
				t.Fatalf("Expected 1 hint diagnostic, got %d", len(hintDiags))
			}

			diag := hintDiags[0]
			if diag.Severity != protocol.DiagnosticSeverityHint {
				t.Errorf("Expected Hint severity, got %v", diag.Severity)
			}

			if diag.Data == nil {
				t.Fatal("Expected diagnostic to have code actions")
			}

			codeActions, ok := diag.Data.([]protocol.CodeAction)
			if !ok {
				t.Fatalf("Expected Data to be []protocol.CodeAction, got %T", diag.Data)
			}

			if len(codeActions) == 0 {
				t.Fatal("Expected at least one code action")
			}

			// Find the terminal simplification code action
			var terminalAction *protocol.CodeAction
			for i := range codeActions {
				if codeActions[i].Title == "Simplify these statuses to 'terminal'" {
					terminalAction = &codeActions[i]
					break
				}
			}

			if terminalAction == nil {
				t.Fatal("Expected 'Simplify these statuses to 'terminal'' code action")
			}

			if terminalAction.Kind != "quickfix" {
				t.Errorf("Expected kind 'quickfix', got %s", terminalAction.Kind)
			}

			if !terminalAction.IsPreferred {
				t.Error("Expected IsPreferred to be true")
			}

			if terminalAction.Edit == nil {
				t.Fatal("Expected Edit to be non-nil")
			}

			changes := terminalAction.Edit.Changes
			if len(changes) != 1 {
				t.Fatalf("Expected 1 change, got %d", len(changes))
			}

			for _, edits := range changes {
				if len(edits) != 1 {
					t.Fatalf("Expected 1 text edit, got %d", len(edits))
				}

				edit := edits[0]
				if edit.NewText != tt.expectedNewText {
					t.Errorf("Expected NewText %q, got %q", tt.expectedNewText, edit.NewText)
				}

				if edit.Range != tt.expectedRange {
					t.Errorf("Expected Range %v, got %v", tt.expectedRange, edit.Range)
				}
			}
		})
	}
}

func TestValidateDAG(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name:       "No cycle - linear chain",
			OnlyErrors: true,
			YamlContent: `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "build"
  test:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"
  deploy:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "deploy"

workflows:
  main:
    jobs:
      - build
      - test:
          requires:
            - build
      - deploy:
          requires:
            - test`,
		},
		{
			Name:       "No cycle - fan out",
			OnlyErrors: true,
			YamlContent: `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "build"
  test:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"
  lint:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "lint"

workflows:
  main:
    jobs:
      - build
      - test:
          requires:
            - build
      - lint:
          requires:
            - build`,
		},
		{
			Name:       "Cycle between two jobs",
			OnlyErrors: true,
			YamlContent: `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "build"
  test:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"

workflows:
  main:
    jobs:
      - build:
          requires:
            - test
      - test:
          requires:
            - build`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 17, Character: 8},
					End:   protocol.Position{Line: 17, Character: 13},
				}, "The job `build` is part of a cycle"),
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 20, Character: 8},
					End:   protocol.Position{Line: 20, Character: 12},
				}, "The job `test` is part of a cycle"),
			},
		},
		{
			Name:       "Indirect cycle - three jobs",
			OnlyErrors: true,
			YamlContent: `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "build"
  test:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"
  deploy:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "deploy"

workflows:
  main:
    jobs:
      - build:
          requires:
            - deploy
      - test:
          requires:
            - build
      - deploy:
          requires:
            - test`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 22, Character: 8},
					End:   protocol.Position{Line: 22, Character: 13},
				}, "The job `build` is part of a cycle"),
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 25, Character: 8},
					End:   protocol.Position{Line: 25, Character: 12},
				}, "The job `test` is part of a cycle"),
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 28, Character: 8},
					End:   protocol.Position{Line: 28, Character: 14},
				}, "The job `deploy` is part of a cycle"),
			},
		},
	}

	CheckYamlErrors(t, testCases)
}
