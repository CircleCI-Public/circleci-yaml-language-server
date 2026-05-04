package languageservice

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/testHelpers"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestFindErrors(t *testing.T) {
	cache := utils.CreateCache()

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, _ := os.ReadFile(tt.args.filePath)
			cache.FileCache.SetFile(utils.CachedFile{
				TextDocument: protocol.TextDocumentItem{
					URI:  uri.File(tt.args.filePath),
					Text: string(content),
				},
				Project:      utils.Project{},
				EnvVariables: make([]string, 0),
			})
			context := testHelpers.GetDefaultLsContext()
			context.Api.Token = ""
			fileUri := uri.File(tt.args.filePath)
			diagnostics, err := DiagnosticFile(fileUri, cache, context, "")

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
	cache := utils.CreateCache()

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, _ := os.ReadFile(tt.filePath)
			cache.FileCache.SetFile(utils.CachedFile{
				TextDocument: protocol.TextDocumentItem{
					URI:  uri.File(tt.filePath),
					Text: string(content),
				},
				Project:      utils.Project{},
				EnvVariables: make([]string, 0),
			})
			context := testHelpers.GetDefaultLsContext()
			context.Api.Token = ""
			fileUri := uri.File(tt.filePath)

			// Pass empty schemaLocation to exercise the embedded schema fallback
			diagnostics, err := DiagnosticFile(fileUri, cache, context, "")

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
			cache := utils.CreateCache()
			content, err := os.ReadFile(tt.filePath)
			if err != nil {
				t.Fatalf("failed to read test file %s: %v", tt.filePath, err)
			}
			cache.FileCache.SetFile(utils.CachedFile{
				TextDocument: protocol.TextDocumentItem{
					URI:  uri.File(tt.filePath),
					Text: string(content),
				},
				Project:      utils.Project{},
				EnvVariables: make([]string, 0),
			})
			context := testHelpers.GetDefaultLsContext()
			context.Api.Token = ""
			fileUri := uri.File(tt.filePath)

			fileDiags, err := DiagnosticFile(fileUri, cache, context, schemaPath)
			if err != nil {
				t.Fatalf("DiagnosticFile() with file schema returned error: %v", err)
			}

			embeddedDiags, err := DiagnosticFile(fileUri, cache, context, "")
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
