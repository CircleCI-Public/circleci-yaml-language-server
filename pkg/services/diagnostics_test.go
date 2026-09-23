package languageservice

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
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
					Message:  "Test error",
					Severity: protocol.DiagnosticSeverityError,
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  "Test error",
					Severity: protocol.DiagnosticSeverityError,
				},
			},
			expected: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  "Test error",
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
					Message:  "Test error",
					Severity: protocol.DiagnosticSeverityError,
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 0},
						End:   protocol.Position{Line: 2, Character: 10},
					},
					Message:  "Test error",
					Severity: protocol.DiagnosticSeverityError,
				},
			},
			expected: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  "Test error",
					Severity: protocol.DiagnosticSeverityError,
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 0},
						End:   protocol.Position{Line: 2, Character: 10},
					},
					Message:  "Test error",
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
					Message:  "Error 1",
					Severity: protocol.DiagnosticSeverityError,
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  "Error 2",
					Severity: protocol.DiagnosticSeverityError,
				},
			},
			expected: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  "Error 1",
					Severity: protocol.DiagnosticSeverityError,
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  "Error 2",
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
					Message:  "Test error",
					Severity: protocol.DiagnosticSeverityError,
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  "Test error",
					Severity: protocol.DiagnosticSeverityWarning,
				},
			},
			expected: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  "Test error",
					Severity: protocol.DiagnosticSeverityError,
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  "Test error",
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
					Message:  "Test error",
					Severity: protocol.DiagnosticSeverityError,
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  "Test error",
					Severity: protocol.DiagnosticSeverityError,
				},
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  "Test error",
					Severity: protocol.DiagnosticSeverityError,
				},
			},
			expected: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 0},
						End:   protocol.Position{Line: 1, Character: 10},
					},
					Message:  "Test error",
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
			for _, diagnostic := range diagnostics {
				if strings.Contains(diagnostic.Message, wanted) {
					return
				}
			}
			t.Errorf("DiagnosticFile() in file %s = %v, want a diagnostic containing %q", filePath, diagnostics, wanted)
		})
	}
}
