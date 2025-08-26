package languageservice

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/testHelpers"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestBackgroundAutoRerunValidation(t *testing.T) {
	cwd, _ := os.Getwd()
	schemaPath, _ := filepath.Abs(cwd + "/../../schema.json")
	os.Setenv("SCHEMA_LOCATION", schemaPath)
	cache := utils.CreateCache()
	context := testHelpers.GetDefaultLsContext()
	context.Api.Token = ""

	testCases := []struct {
		name            string
		yamlContent     string
		expectErrors    bool
		errorSubstrings []string // Substrings that should appear in error messages
	}{
		{
			name: "Valid: background true without auto-rerun fields",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Background task"
          command: "sleep 30"
          background: true
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors: false,
		},
		{
			name: "Valid: max_auto_reruns without background",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with auto reruns"
          command: "echo test"
          max_auto_reruns: 3
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors: false,
		},
		{
			name: "Valid: max_auto_reruns with auto_rerun_delay",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with both auto-rerun fields"
          command: "echo test"
          max_auto_reruns: 2
          auto_rerun_delay: 3m
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors: false,
		},
		{
			name: "Valid: max_auto_reruns with background false",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with background false and max_auto_reruns"
          command: "echo test"
          background: false
          max_auto_reruns: 3
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors: false,
		},
		{
			name: "Valid: both auto-rerun fields with background false",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with background false and both auto-rerun fields"
          command: "echo test"
          background: false
          max_auto_reruns: 2
          auto_rerun_delay: 4m
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors: false,
		},
		{
			name: "Invalid: background true with max_auto_reruns",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Background task with auto reruns"
          command: "sleep 30"
          background: true
          max_auto_reruns: 3
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors:    true,
			errorSubstrings: []string{"Background", "max_auto_reruns"},
		},
		{
			name: "Invalid: background true with auto_rerun_delay",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Background task with auto rerun delay"
          command: "sleep 30"
          background: true
          auto_rerun_delay: 5m
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors:    true,
			errorSubstrings: []string{"auto_rerun_delay"},
		},
		{
			name: "Invalid: auto_rerun_delay without max_auto_reruns",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with auto rerun delay only"
          command: "echo test"
          auto_rerun_delay: 3m
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors:    true,
			errorSubstrings: []string{"auto_rerun_delay", "max_auto_reruns"},
		},
		{
			name: "Valid: max_auto_reruns minimum value (1)",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with minimum auto reruns"
          command: "echo test"
          max_auto_reruns: 1
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors: false,
		},
		{
			name: "Valid: max_auto_reruns maximum value (5)",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with maximum auto reruns"
          command: "echo test"
          max_auto_reruns: 5
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors: false,
		},
		{
			name: "Invalid: max_auto_reruns below minimum (0)",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with zero auto reruns"
          command: "echo test"
          max_auto_reruns: 0
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors:    true,
			errorSubstrings: []string{"max_auto_reruns must be between 1 and 5"},
		},
		{
			name: "Invalid: max_auto_reruns above maximum (6)",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with too many auto reruns"
          command: "echo test"
          max_auto_reruns: 6
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors:    true,
			errorSubstrings: []string{"max_auto_reruns must be between 1 and 5"},
		},
		{
			name: "Valid: auto_rerun_delay with seconds",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with delay in seconds"
          command: "echo test"
          max_auto_reruns: 2
          auto_rerun_delay: 30s
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors: false,
		},
		{
			name: "Valid: auto_rerun_delay with minutes",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with delay in minutes"
          command: "echo test"
          max_auto_reruns: 3
          auto_rerun_delay: 5m
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors: false,
		},
		{
			name: "Invalid: auto_rerun_delay with milliseconds",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with delay in milliseconds"
          command: "echo test"
          max_auto_reruns: 2
          auto_rerun_delay: 500ms
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors:    true,
			errorSubstrings: []string{"auto_rerun_delay must be in the format"},
		},
		{
			name: "Valid: auto_rerun_delay at maximum (10m)",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with maximum delay"
          command: "echo test"
          max_auto_reruns: 2
          auto_rerun_delay: 10m
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors: false,
		},
		{
			name: "Valid: auto_rerun_delay in seconds (600s = 10m)",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with delay in seconds"
          command: "echo test"
          max_auto_reruns: 1
          auto_rerun_delay: 600s
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors: false,
		},
		{
			name: "Invalid: auto_rerun_delay exceeds maximum (11m)",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with delay over maximum"
          command: "echo test"
          max_auto_reruns: 2
          auto_rerun_delay: 11m
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors:    true,
			errorSubstrings: []string{"auto_rerun_delay must not exceed 10 minutes"},
		},
		{
			name: "Invalid: auto_rerun_delay exceeds maximum (700s)",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with delay over 10m in seconds"
          command: "echo test"
          max_auto_reruns: 2
          auto_rerun_delay: 700s
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors:    true,
			errorSubstrings: []string{"auto_rerun_delay must not exceed 10 minutes"},
		},
		{
			name: "Invalid: auto_rerun_delay with invalid format (no unit)",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with invalid delay format"
          command: "echo test"
          max_auto_reruns: 2
          auto_rerun_delay: 30
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors:    true,
			errorSubstrings: []string{"auto_rerun_delay must be a valid duration"},
		},
		{
			name: "Invalid: auto_rerun_delay with invalid text",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with invalid delay text"
          command: "echo test"
          max_auto_reruns: 1
          auto_rerun_delay: invalid
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors:    true,
			errorSubstrings: []string{"auto_rerun_delay must be a valid duration"},
		},
		{
			name: "Invalid: auto_rerun_delay with 0 seconds",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with 0 seconds delay"
          command: "echo test"
          max_auto_reruns: 2
          auto_rerun_delay: 0s
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors:    true,
			errorSubstrings: []string{"auto_rerun_delay must be in the format"},
		},
		{
			name: "Invalid: auto_rerun_delay with 0 minutes",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with 0 minutes delay"
          command: "echo test"
          max_auto_reruns: 2
          auto_rerun_delay: 0m
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors:    true,
			errorSubstrings: []string{"auto_rerun_delay must be in the format"},
		},
		{
			name: "Invalid: auto_rerun_delay with both minutes and seconds",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with both minutes and seconds"
          command: "echo test"
          max_auto_reruns: 2
          auto_rerun_delay: 1m30s
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors:    true,
			errorSubstrings: []string{"auto_rerun_delay must be in the format"},
		},
		{
			name: "Valid: auto_rerun_delay edge case (1s)",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with minimum delay"
          command: "echo test"
          max_auto_reruns: 1
          auto_rerun_delay: 1s
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors: false,
		},
		{
			name: "Valid: auto_rerun_delay edge case (9m)",
			yamlContent: `version: 2.1
jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Task with 9 minute delay"
          command: "echo test"
          max_auto_reruns: 2
          auto_rerun_delay: 9m
workflows:
  test-workflow:
    jobs:
      - test-job`,
			expectErrors: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a temporary file for the test
			testUri := uri.File("test.yml")
			cache.FileCache.SetFile(utils.CachedFile{
				TextDocument: protocol.TextDocumentItem{
					URI:  testUri,
					Text: tc.yamlContent,
				},
				Project:      utils.Project{},
				EnvVariables: make([]string, 0),
			})

			diagnostics, err := DiagnosticFile(testUri, cache, context, schemaPath)
			if err != nil {
				t.Errorf("DiagnosticFile failed: %v", err)
				return
			}

			// Filter to only error-level diagnostics
			errorDiagnostics := []protocol.Diagnostic{}
			for _, diag := range diagnostics {
				if diag.Severity == protocol.DiagnosticSeverityError {
					errorDiagnostics = append(errorDiagnostics, diag)
				}
			}

			if tc.expectErrors {
				if len(errorDiagnostics) == 0 {
					t.Errorf("Expected errors but got none")
					return
				}

				// Check that error messages contain expected substrings
				for _, expectedSubstring := range tc.errorSubstrings {
					found := false
					for _, diag := range errorDiagnostics {
						if contains(diag.Message, expectedSubstring) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected error message to contain '%s', but no error contained it. Errors: %v",
							expectedSubstring, getErrorMessages(errorDiagnostics))
					}
				}
			} else {
				if len(errorDiagnostics) > 0 {
					t.Errorf("Expected no errors but got: %v", getErrorMessages(errorDiagnostics))
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Helper function to extract error messages for debugging
func getErrorMessages(diagnostics []protocol.Diagnostic) []string {
	messages := make([]string, len(diagnostics))
	for i, diag := range diagnostics {
		messages[i] = diag.Message
	}
	return messages
}
