package languageservice

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestFindErrors(t *testing.T) {
	c := cache.New()

	type args struct {
		filePath string
	}
	tests := []struct {
		name string
		args args
		want []protocol.Diagnostic
	}{
		{
			name: "No errors",
			args: args{filePath: "./testdata/noErrors.yml"},
			want: make([]protocol.Diagnostic, 0),
		},
		{
			name: "No errors",
			args: args{filePath: "./testdata/anchorNoErrors.yml"},
			want: make([]protocol.Diagnostic, 0),
		},
		{
			name: "No errors",
			args: args{filePath: "./testdata/requiresNoErrors.yml"},
			want: make([]protocol.Diagnostic, 0),
		},
		{
			name: "No errors",
			args: args{filePath: "./testdata/stepWhen.yml"},
			want: make([]protocol.Diagnostic, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, _ := os.ReadFile(tt.args.filePath)
			c.FileCache.SetFile(cache.File{
				TextDocument: protocol.TextDocumentItem{
					URI:  uri.File(tt.args.filePath),
					Text: string(content),
				},
				Project:      circleci.Project{},
				EnvVariables: make([]string, 0),
			})
			context := testHelpers.DefaultSettings()
			context.Api.Token = ""
			fileUri := uri.File(tt.args.filePath)
			diagnostics, err := DiagnosticFile(fileUri, c, context, "")

			if err != nil {
				t.Error("findErrors()", err)
			}

			if !reflect.DeepEqual(diagnostics, tt.want) {
				t.Errorf("FindErrors() in file %s = %v, want %v", tt.args.filePath, diagnostics, tt.want)
			}
		})
	}
}

func TestFindErrorsWithEmbeddedSchema(t *testing.T) {
	c := cache.New()

	tests := []struct {
		name     string
		filePath string
		want     []protocol.Diagnostic
	}{
		{
			name:     "No errors with embedded schema",
			filePath: "./testdata/noErrors.yml",
			want:     make([]protocol.Diagnostic, 0),
		},
		{
			name:     "when attribute on every built-in step with embedded schema",
			filePath: "./testdata/stepWhen.yml",
			want:     make([]protocol.Diagnostic, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, _ := os.ReadFile(tt.filePath)
			c.FileCache.SetFile(cache.File{
				TextDocument: protocol.TextDocumentItem{
					URI:  uri.File(tt.filePath),
					Text: string(content),
				},
				Project:      circleci.Project{},
				EnvVariables: make([]string, 0),
			})
			context := testHelpers.DefaultSettings()
			context.Api.Token = ""
			fileUri := uri.File(tt.filePath)

			// Pass empty schemaLocation to exercise the embedded schema fallback
			diagnostics, err := DiagnosticFile(fileUri, c, context, "")

			if err != nil {
				t.Fatalf("DiagnosticFile() with embedded schema returned error: %v", err)
			}

			if !reflect.DeepEqual(diagnostics, tt.want) {
				t.Errorf("DiagnosticFile() with embedded schema in file %s = %v, want %v", tt.filePath, diagnostics, tt.want)
			}
		})
	}
}

func TestOverrideSchemaMatchesEmbeddedSchema(t *testing.T) {
	cwd, _ := os.Getwd()
	schemaPath := filepath.Join(cwd, "..", "..", "schema.json")

	tests := []struct {
		name           string
		filePath       string
		expectNonEmpty bool
	}{
		{
			name:           "Clean file produces no diagnostics from either schema",
			filePath:       "./testdata/noErrors.yml",
			expectNonEmpty: false,
		},
		{
			name:           "File with schema error produces matching diagnostics",
			filePath:       "./testdata/schemaError.yml",
			expectNonEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := cache.New()
			content, err := os.ReadFile(tt.filePath)
			if err != nil {
				t.Fatalf("failed to read test file %s: %v", tt.filePath, err)
			}
			c.FileCache.SetFile(cache.File{
				TextDocument: protocol.TextDocumentItem{
					URI:  uri.File(tt.filePath),
					Text: string(content),
				},
				Project:      circleci.Project{},
				EnvVariables: make([]string, 0),
			})
			context := testHelpers.DefaultSettings()
			context.Api.Token = ""
			fileUri := uri.File(tt.filePath)

			fileDiags, err := DiagnosticFile(fileUri, c, context, schemaPath)
			if err != nil {
				t.Fatalf("DiagnosticFile() with file schema returned error: %v", err)
			}

			embeddedDiags, err := DiagnosticFile(fileUri, c, context, "")
			if err != nil {
				t.Fatalf("DiagnosticFile() with embedded schema returned error: %v", err)
			}

			if tt.expectNonEmpty && len(fileDiags) == 0 {
				t.Error("expected diagnostics from file schema but got none")
			}

			if !reflect.DeepEqual(fileDiags, embeddedDiags) {
				t.Errorf("Override schema produced different diagnostics than embedded schema.\nFile: %v\nEmbedded: %v", fileDiags, embeddedDiags)
			}
		})
	}
}

func TestDeduplicateDiagnosticsByRange(t *testing.T) {
	tests := []struct {
		name     string
		input    []protocol.Diagnostic
		expected []protocol.Diagnostic
	}{
		{
			name: "Remove exact duplicates",
			input: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  protocol.String("Test error"),
					Severity: protocol.DiagnosticSeverityError,
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  protocol.String("Test error"),
					Severity: protocol.DiagnosticSeverityError,
				},
			},
			expected: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  protocol.String("Test error"),
					Severity: protocol.DiagnosticSeverityError,
				},
			},
		},
		{
			name: "Keep different ranges",
			input: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  protocol.String("Test error"),
					Severity: protocol.DiagnosticSeverityError,
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 0},
						End:   protocol.Position{Line: 2, Character: 10},
					},
					Message:  protocol.String("Test error"),
					Severity: protocol.DiagnosticSeverityError,
				},
			},
			expected: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  protocol.String("Test error"),
					Severity: protocol.DiagnosticSeverityError,
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 0},
						End:   protocol.Position{Line: 2, Character: 10},
					},
					Message:  protocol.String("Test error"),
					Severity: protocol.DiagnosticSeverityError,
				},
			},
		},
		{
			name: "Keep same range different messages",
			input: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  protocol.String("Error 1"),
					Severity: protocol.DiagnosticSeverityError,
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  protocol.String("Error 2"),
					Severity: protocol.DiagnosticSeverityError,
				},
			},
			expected: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  protocol.String("Error 1"),
					Severity: protocol.DiagnosticSeverityError,
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  protocol.String("Error 2"),
					Severity: protocol.DiagnosticSeverityError,
				},
			},
		},
		{
			name: "Keep same range different severities",
			input: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  protocol.String("Test error"),
					Severity: protocol.DiagnosticSeverityError,
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  protocol.String("Test error"),
					Severity: protocol.DiagnosticSeverityWarning,
				},
			},
			expected: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  protocol.String("Test error"),
					Severity: protocol.DiagnosticSeverityError,
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  protocol.String("Test error"),
					Severity: protocol.DiagnosticSeverityWarning,
				},
			},
		},
		{
			name: "Remove multiple duplicates",
			input: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  protocol.String("Test error"),
					Severity: protocol.DiagnosticSeverityError,
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  protocol.String("Test error"),
					Severity: protocol.DiagnosticSeverityError,
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  protocol.String("Test error"),
					Severity: protocol.DiagnosticSeverityError,
				},
			},
			expected: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  protocol.String("Test error"),
					Severity: protocol.DiagnosticSeverityError,
				},
			},
		},
		{
			name:     "Empty input",
			input:    []protocol.Diagnostic{},
			expected: []protocol.Diagnostic{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deduplicateDiagnosticsByRange(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("deduplicateDiagnosticsByRange() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestStepWhenRejectsInvalidValue(t *testing.T) {
	const filePath = "./testdata/stepWhenInvalid.yml"

	stepNames := []string{
		"checkout",
		"setup_remote_docker",
		"add_ssh_keys",
		"restore_cache",
		"run",
		"save_cache",
		"store_artifacts",
		"store_test_results",
		"persist_to_workspace",
		"attach_workspace",
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	c := cache.New()
	c.FileCache.SetFile(cache.File{
		TextDocument: protocol.TextDocumentItem{
			URI:  uri.File(filePath),
			Text: string(content),
		},
		Project:      circleci.Project{},
		EnvVariables: make([]string, 0),
	})
	context := testHelpers.DefaultSettings()
	context.Api.Token = ""

	diagnostics, err := DiagnosticFile(uri.File(filePath), c, context, "")
	if err != nil {
		t.Fatalf("DiagnosticFile() returned error: %v", err)
	}

	for _, stepName := range stepNames {
		t.Run(stepName, func(t *testing.T) {
			wanted := "." + stepName + `.when must be one of the following: "always", "on_success", "on_fail"`
			for _, d := range diagnostics {
				if strings.Contains(diagnostic.MessageText(d), wanted) {
					return
				}
			}
			t.Errorf("DiagnosticFile() in file %s = %v, want a diagnostic containing %q", filePath, diagnostics, wanted)
		})
	}
}

func TestFilesThatAreNotPipelineConfig(t *testing.T) {
	diagnose := func(t *testing.T, fileName, content string) []protocol.Diagnostic {
		t.Helper()

		fileURI := uri.File(filepath.Join(t.TempDir(), ".circleci", fileName))
		c := cache.New()
		c.FileCache.SetFile(cache.File{
			TextDocument: protocol.TextDocumentItem{URI: fileURI, Text: content},
		})
		settings := testHelpers.DefaultSettings()
		settings.Api.Token = ""

		diagnostics, err := DiagnosticFile(fileURI, c, settings, "")
		assert.NilError(t, err)
		return diagnostics
	}

	testSuites := `name: ci tests
discover: go list ./...
run: gotestsum -- << test.atoms >>
outputs:
  junit: test-reports/tests.xml
---
name: windows
run: gotestsum -- ./...
`

	t.Run("a Smarter Testing definition is not validated", func(t *testing.T) {
		assert.Check(t, cmp.Len(diagnose(t, "test-suites.yml", testSuites), 0))
	})

	t.Run("a file for another tool is not validated", func(t *testing.T) {
		assert.Check(t, cmp.Len(diagnose(t, "factory-bot.yml", "review:\n  enabled: true\n"), 0))
	})

	t.Run("config.yml is validated whatever it holds", func(t *testing.T) {
		assert.Check(t, len(diagnose(t, "config.yml", "review:\n  enabled: true\n")) != 0)
	})

	t.Run("a file with a pipeline key is validated", func(t *testing.T) {
		assert.Check(t, len(diagnose(t, "continue.yml", "jobs:\n  build: {}\n")) != 0)
	})
}

// Each case is config the compiler accepts, so the schema mustn't flag it.
func TestSchemaAcceptsWhatTheCompilerAccepts(t *testing.T) {
	fake := fakes.NewCircleCI(t)

	errors := func(t *testing.T, content string) []string {
		t.Helper()

		fileURI := uri.File(filepath.Join(t.TempDir(), ".circleci", "config.yml"))
		c := cache.New()
		c.FileCache.SetFile(cache.File{
			TextDocument: protocol.TextDocumentItem{URI: fileURI, Text: content},
		})

		diagnostics, err := DiagnosticFile(fileURI, c, testHelpers.SettingsForHost(fake.URL()), "")
		assert.NilError(t, err)

		messages := []string{}
		for _, d := range diagnostics {
			if d.Severity == protocol.DiagnosticSeverityError {
				messages = append(messages, diagnostic.MessageText(d))
			}
		}
		return messages
	}

	job := func(steps string) string {
		return `version: 2.1
jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
` + steps + `
workflows:
  main:
    jobs:
      - build
`
	}

	tests := []struct {
		name    string
		content string
	}{
		{
			name: "save_cache paths as a string",
			content: job(`      - save_cache:
          key: deps
          paths: /virtualenvs`),
		},
		{
			name: "restore_cache keys as a string",
			content: job(`      - restore_cache:
          keys: deps`),
		},
		{
			name: "persist_to_workspace paths as a string",
			content: job(`      - persist_to_workspace:
          root: .
          paths: bin`),
		},
		{
			name: "persist_to_workspace with a single path",
			content: job(`      - persist_to_workspace:
          root: .
          path: bin`),
		},
		{
			name: "add_ssh_keys fingerprints as a string",
			content: job(`      - add_ssh_keys:
          fingerprints: "SHA256:abc"`),
		},
		{
			name: "an executor resource class the schema doesn't list",
			content: `version: 2.1
executors:
  gen3:
    docker:
      - image: cimg/base:current
    resource_class: large.gen3
jobs:
  build:
    executor: gen3
    steps:
      - checkout
workflows:
  main:
    jobs:
      - build
`,
		},
		{
			name: "an executor resource class from a parameter",
			content: `version: 2.1
executors:
  sized:
    parameters:
      resource_class:
        type: string
        default: medium
    docker:
      - image: cimg/base:current
    resource_class: <<parameters.resource_class>>
jobs:
  build:
    executor: sized
    steps:
      - checkout
workflows:
  main:
    jobs:
      - build
`,
		},
		{
			name: "an orb's development version at the commit being built",
			content: `version: 2.1
orbs:
  shellcheck: circleci/shellcheck@dev:<<pipeline.git.revision>>
jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - checkout
workflows:
  main:
    jobs:
      - build
`,
		},
		{
			name: "expressions in typed fields",
			content: `version: 2.1
parameters:
  reruns:
    type: integer
    default: 2
  delay:
    type: string
    default: 30s
jobs:
  build:
    parameters:
      split:
        type: integer
        default: 2
      fixed_ips:
        type: boolean
        default: false
    docker:
      - image: cimg/base:current
    parallelism: << matrix.split >>
    circleci_ip_ranges: << parameters.fixed_ips >>
    steps:
      - run:
          command: make test
          max_auto_reruns: << pipeline.parameters.reruns >>
          auto_rerun_delay: << pipeline.parameters.delay >>
workflows:
  main:
    max_auto_reruns: << pipeline.parameters.reruns >>
    jobs:
      - build:
          matrix:
            parameters:
              split: [2, 4]
`,
		},
		{
			name: "no_output_timeout as an integer or an expression",
			content: `version: 2.1
jobs:
  build:
    parameters:
      timeout:
        type: string
        default: 20m
    docker:
      - image: cimg/base:current
    steps:
      - run:
          command: make test
          no_output_timeout: 600
      - run:
          command: make test
          no_output_timeout: << parameters.timeout >>
workflows:
  main:
    jobs:
      - build
`,
		},
		{
			name: "code signing bundles on a machine executor",
			content: `version: 2.1
executors:
  signing:
    machine:
      image: ubuntu-2204:current
      code_signing:
        - release-bundle
jobs:
  build:
    executor: signing
    steps:
      - checkout
workflows:
  main:
    jobs:
      - build
`,
		},
		{
			name: "caches kept for 30 days",
			content: `version: 2.1
jobs:
  build:
    docker:
      - image: cimg/base:current
    retention:
      caches: 30d
    steps:
      - checkout
workflows:
  main:
    jobs:
      - build
`,
		},
		{
			name: "experimental without notify",
			content: `version: 2.1
experimental:
  observability: {}
jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - checkout
workflows:
  main:
    jobs:
      - build
`,
		},
		{
			name: "an image pulled with GCP OIDC",
			content: `version: 2.1
jobs:
  build:
    docker:
      - image: us-docker.pkg.dev/my-project/images/builder:1.0
        gcp_auth:
          oidc_service_account: builder@my-project.iam.gserviceaccount.com
          workload_identity_pool: circleci
          workload_identity_provider: circleci-oidc
    steps:
      - checkout
workflows:
  main:
    jobs:
      - build
`,
		},
		{
			name: "setup_remote_docker options the schema didn't know",
			content: job(`      - setup_remote_docker:
          version: default
          prefer_same_region: true
      - setup_remote_docker:
          resource_class: large`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Check(t, cmp.DeepEqual(errors(t, tt.content), []string{}))
		})
	}
}
