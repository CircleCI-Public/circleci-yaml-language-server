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
				}, "Only jobs with `type: approval` can be defined inline under the `workflows:`/`job-groups:` section. For `type: invalid`, define the job in the `jobs:` section instead."),
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

func TestNestedJobGroups(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "nested job groups are not allowed",
			YamlContent: `version: 2.1

jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"

job-groups:
  my-group1:
    jobs:
      - test-job
  my-group2:
    jobs:
      - my-group1

workflows:
  test-workflow:
    jobs:
      - my-group1`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 15, Character: 8},
						End:   protocol.Position{Line: 15, Character: 17},
					},
					Severity: protocol.DiagnosticSeverityError,
					Source:   "cci-language-server",
					Message:  `Job group "my-group2" cannot reference job group "my-group1" -- nesting is not supported`,
					Data:     []protocol.CodeAction{},
				},
			},
		},
	}

	CheckYamlErrors(t, testCases)
}

func TestJobInvocationNonExistentJob(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "Job invocation referencing a non-existent job/job-group produces an error diagnostic",
			YamlContent: `version: 2.1

workflows:
  test-workflow:
    jobs:
      - my-ghost-job`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 5, Character: 6},
						End:   protocol.Position{Line: 5, Character: 20},
					},
					Severity: protocol.DiagnosticSeverityError,
					Source:   "cci-language-server",
					Message:  "Cannot find declaration for job \"my-ghost-job\"",
					Data:     []protocol.CodeAction{},
				},
			},
		},
		{
			Name: "Job invocation referencing a non-existent orb job produces an error diagnostic",
			YamlContent: `version: 2.1

workflows:
  test-workflow:
    jobs:
      - my-orb/my-ghost-job`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 5, Character: 6},
						End:   protocol.Position{Line: 5, Character: 27},
					},
					Severity: protocol.DiagnosticSeverityError,
					Source:   "cci-language-server",
					Message:  "Cannot find declaration for job \"my-orb/my-ghost-job\"",
					Data:     []protocol.CodeAction{},
				},
			},
		},
	}

	CheckYamlErrors(t, testCases)
}

func TestJobInvocationRequiresNonExistentRef(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "Requires references a job that does not exist in invocations",
			YamlContent: `version: 2.1

jobs:
  test:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"

workflows:
  test-workflow:
    jobs:
      - test:
          requires:
            - ghost-job`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 14, Character: 14},
					End:   protocol.Position{Line: 14, Character: 23},
				}, "Cannot find declaration for job invocation \"ghost-job\""),
			},
		},
		{
			Name: "Requires with matrix partial reference is allowed",
			YamlContent: `version: 2.1

jobs:
  build:
    parameters:
      os:
        type: string
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo <<parameters.os>>
  test:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"

workflows:
  test-workflow:
    jobs:
      - build:
          matrix:
            parameters:
              os: [linux, macos]
      - test:
          requires:
            - build-<< matrix.os >>`,
			OnlyErrors: true,
		},
	}

	CheckYamlErrors(t, testCases)
}

func TestJobInvocationExistsByStepName(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "Requires can reference a job by its step name alias",
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
  test-workflow:
    jobs:
      - build:
          name: my-build-alias
      - test:
          requires:
            - my-build-alias`,
			OnlyErrors: true,
		},
	}

	CheckYamlErrors(t, testCases)
}

func TestJobInvocationInJobGroupContext(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "Job invocation validation works inside a job-group",
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

job-groups:
  ci-group:
    jobs:
      - build
      - test:
          requires:
            - build

workflows:
  main:
    jobs:
      - ci-group`,
			OnlyErrors: true,
		},
		{
			Name: "Non-existent job in job-group produces error",
			YamlContent: `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "build"

job-groups:
  ci-group:
    jobs:
      - build
      - ghost-job

workflows:
  main:
    jobs:
      - ci-group`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 13, Character: 6},
					End:   protocol.Position{Line: 13, Character: 17},
				}, `Cannot find declaration for job "ghost-job"`),
			},
		},
	}

	CheckYamlErrors(t, testCases)
}

func TestRequiresJobGroupMember_FromWorkflow(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name:       "workflow job requires a member of a job-group",
			OnlyErrors: true,
			YamlContent: `version: 2.1

jobs:
  deploy:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "deploy"
  after:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "after"

job-groups:
  deploy-group:
    jobs:
      - deploy

workflows:
  main:
    jobs:
      - deploy-group
      - after:
          requires:
            - deploy`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 25, Character: 14},
					End:   protocol.Position{Line: 25, Character: 20},
				}, `"deploy" is defined inside job group "deploy-group", not directly in this workflow`),
			},
		},
	}

	CheckYamlErrors(t, testCases)
}

func TestRequiresJobGroupMember_FromDifferentGroup(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name:       "job-group member requires a job from a different group",
			OnlyErrors: true,
			YamlContent: `version: 2.1

jobs:
  a:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "a"
  b:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "b"

job-groups:
  g1:
    jobs:
      - a
  g2:
    jobs:
      - b:
          requires:
            - a

workflows:
  main:
    jobs:
      - g1
      - g2`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 22, Character: 14},
					End:   protocol.Position{Line: 22, Character: 15},
				}, `"a" is not a member of this job group`),
			},
		},
	}

	CheckYamlErrors(t, testCases)
}

func TestRequiresJobGroupMember_SameGroupIsValid(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name:       "jobs within the same group can require each other",
			OnlyErrors: true,
			YamlContent: `version: 2.1

jobs:
  deploy:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "deploy"
  release:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "release"

job-groups:
  deploy-group:
    jobs:
      - deploy
      - release:
          requires:
            - deploy

workflows:
  main:
    jobs:
      - deploy-group`,
		},
	}

	CheckYamlErrors(t, testCases)
}

func TestRequiresJobGroup_FromWorkflowIsValid(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name:       "requiring a job-group name from a workflow is valid",
			OnlyErrors: true,
			YamlContent: `version: 2.1

jobs:
  deploy:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "deploy"
  notify:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "notify"

job-groups:
  deploy-group:
    jobs:
      - deploy

workflows:
  main:
    jobs:
      - deploy-group
      - notify:
          requires:
            - deploy-group`,
		},
	}

	CheckYamlErrors(t, testCases)
}

func TestValidateSingleJobInvocation_SerialGroups(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name:       "serial-group in workflow is allowed",
			OnlyErrors: true,
			YamlContent: `version: 2.1

jobs:
  deploy:
    type: no-op

workflows:
  test-workflow:
    jobs:
      - deploy:
          serial-group: deploy-group`,
		},
		{
			Name:       "serial-group in job-group is not allowed",
			OnlyErrors: true,
			YamlContent: `version: 2.1

jobs:
  deploy:
    type: no-op

job-groups:
  my-group:
    jobs:
      - deploy:
          serial-group: deploy-group

workflows:
  test-workflow:
    jobs:
      - my-group`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 10, Character: 24},
					End:   protocol.Position{Line: 10, Character: 36},
				}, "Use of `serial-group` on job invocations inside a job-group is not supported. Please consider using `serial-group` on the job-group instead."),
			},
		},
	}

	CheckYamlErrors(t, testCases)
}

func TestJobGroupInvocation_NameAttribute(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name:       "job-group invocation with name: is allowed and can be required by that name",
			OnlyErrors: true,
			YamlContent: `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "build"
  deploy:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "deploy"
  notify:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "notify"

job-groups:
  deploy-group:
    jobs:
      - deploy

workflows:
  main:
    jobs:
      - build
      - deploy-group:
          name: prod-deploy
          requires:
            - build
      - notify:
          requires:
            - prod-deploy`,
		},
		{
			Name:       "same job-group invoked twice with different names is valid",
			OnlyErrors: true,
			YamlContent: `version: 2.1

jobs:
  deploy:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "deploy"

job-groups:
  deploy-group:
    jobs:
      - deploy

workflows:
  main:
    jobs:
      - deploy-group:
          name: staging-deploy
      - deploy-group:
          name: prod-deploy
          requires:
            - staging-deploy`,
		},
		{
			Name:       "same job-group invoked twice with same name is an error",
			OnlyErrors: true,
			YamlContent: `version: 2.1

jobs:
  deploy:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "deploy"

job-groups:
  deploy-group:
    jobs:
      - deploy

workflows:
  main:
    jobs:
      - deploy-group:
          name: prod-deploy
      - deploy-group:
          name: prod-deploy`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 20, Character: 16},
					End:   protocol.Position{Line: 20, Character: 27},
				}, `Job group "deploy-group" is already invoked with the name "prod-deploy"`),
			},
		},
		{
			Name:       "same job-group invoked twice without name is an error on both",
			OnlyErrors: true,
			YamlContent: `version: 2.1

jobs:
  deploy:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "deploy"

job-groups:
  deploy-group:
    jobs:
      - deploy

workflows:
  main:
    jobs:
      - deploy-group
      - deploy-group`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 17, Character: 8},
					End:   protocol.Position{Line: 17, Character: 20},
				}, `Job group "deploy-group" is invoked multiple times without a "name" attribute. Each invocation must have a unique name`),
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 18, Character: 8},
					End:   protocol.Position{Line: 18, Character: 20},
				}, `Job group "deploy-group" is invoked multiple times without a "name" attribute. Each invocation must have a unique name`),
			},
		},
		{
			Name:       "one invocation without name, one with name is an error (the nameless one)",
			OnlyErrors: true,
			YamlContent: `version: 2.1

jobs:
  deploy:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "deploy"

job-groups:
  deploy-group:
    jobs:
      - deploy

workflows:
  main:
    jobs:
      - deploy-group
      - deploy-group:
          name: prod-deploy`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 17, Character: 8},
					End:   protocol.Position{Line: 17, Character: 20},
				}, `Job group "deploy-group" is invoked multiple times without a "name" attribute. Each invocation must have a unique name`),
			},
		},
	}

	CheckYamlErrors(t, testCases)
}

func TestJobGroupInvocationDisallowedKeys(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name:       "matrix on job-group invocation is not allowed",
			OnlyErrors: true,
			YamlContent: `version: 2.1

jobs:
  deploy:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "deploy"

job-groups:
  deploy-group:
    jobs:
      - deploy

workflows:
  main:
    jobs:
      - deploy-group:
          matrix:
            parameters:
              env: [staging, prod]`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 18, Character: 10},
					End:   protocol.Position{Line: 20, Character: 34},
				}, "Job group invocations do not support `matrix`"),
			},
		},
		{
			Name:       "override-with on job-group invocation is not allowed",
			OnlyErrors: true,
			YamlContent: `version: 2.1

jobs:
  deploy:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "deploy"

job-groups:
  deploy-group:
    jobs:
      - deploy

workflows:
  main:
    jobs:
      - deploy-group:
          override-with: other-job`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 18, Character: 25},
					End:   protocol.Position{Line: 18, Character: 34},
				}, "Job group invocations do not support use of `override-with`"),
			},
		},
		{
			Name:       "type on job-group invocation is not allowed",
			OnlyErrors: true,
			YamlContent: `version: 2.1

jobs:
  deploy:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "deploy"

job-groups:
  deploy-group:
    jobs:
      - deploy

workflows:
  main:
    jobs:
      - deploy-group:
          type: approval`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 18, Character: 16},
					End:   protocol.Position{Line: 18, Character: 24},
				}, "Job group invocations do not support use of `type`"),
			},
		},
		{
			Name:       "context on job-group invocation is not allowed",
			OnlyErrors: true,
			YamlContent: `version: 2.1

jobs:
  deploy:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "deploy"

job-groups:
  deploy-group:
    jobs:
      - deploy

workflows:
  main:
    jobs:
      - deploy-group:
          context:
            - my-context`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 17, Character: 6},
					End:   protocol.Position{Line: 19, Character: 24},
				}, "Job group invocations do not support use of `context`"),
			},
		},
		{
			Name:       "pre-steps on job-group invocation is not allowed",
			OnlyErrors: true,
			YamlContent: `version: 2.1

jobs:
  deploy:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "deploy"

job-groups:
  deploy-group:
    jobs:
      - deploy

workflows:
  main:
    jobs:
      - deploy-group:
          pre-steps:
            - run: echo "pre"`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 18, Character: 10},
					End:   protocol.Position{Line: 19, Character: 29},
				}, "Job group invocations do not support use of `pre-steps`"),
			},
		},
		{
			Name:       "post-steps on job-group invocation is not allowed",
			OnlyErrors: true,
			YamlContent: `version: 2.1

jobs:
  deploy:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "deploy"

job-groups:
  deploy-group:
    jobs:
      - deploy

workflows:
  main:
    jobs:
      - deploy-group:
          post-steps:
            - run: echo "post"`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 18, Character: 10},
					End:   protocol.Position{Line: 19, Character: 30},
				}, "Job group invocations do not support use of `post-steps`"),
			},
		},
		{
			Name:       "custom parameters on job-group invocation is not allowed",
			OnlyErrors: true,
			YamlContent: `version: 2.1

jobs:
  deploy:
    parameters:
      env:
        type: string
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo <<parameters.env>>

job-groups:
  deploy-group:
    jobs:
      - deploy

workflows:
  main:
    jobs:
      - deploy-group:
          env: staging`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 20, Character: 6},
					End:   protocol.Position{Line: 21, Character: 22},
				}, "Job group invocations do not support custom parameters, but found: `env`"),
			},
		},
		{
			Name:       "requires and serial-group on job-group invocation are allowed",
			OnlyErrors: true,
			YamlContent: `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "build"
  deploy:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "deploy"

job-groups:
  deploy-group:
    jobs:
      - deploy

workflows:
  main:
    jobs:
      - build
      - deploy-group:
          serial-group: my-serial
          requires:
            - build`,
		},
		{
			Name:       "multiple disallowed keys on job-group invocation produce multiple errors",
			OnlyErrors: true,
			YamlContent: `version: 2.1

jobs:
  deploy:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "deploy"

job-groups:
  deploy-group:
    jobs:
      - deploy

workflows:
  main:
    jobs:
      - deploy-group:
          type: approval
          context:
            - my-context`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 17, Character: 6},
					End:   protocol.Position{Line: 20, Character: 24},
				}, "Job group invocations do not support use of `context`"),
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 18, Character: 16},
					End:   protocol.Position{Line: 18, Character: 24},
				}, "Job group invocations do not support use of `type`"),
			},
		},
	}

	CheckYamlErrors(t, testCases)
}
