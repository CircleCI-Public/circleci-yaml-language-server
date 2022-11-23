package languageservice

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestFindErrors(t *testing.T) {
	cwd, _ := os.Getwd()
	schemaPath, _ := filepath.Abs(cwd + "/../../schema.json")
	os.Setenv("SCHEMA_LOCATION", schemaPath)
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, _ := os.ReadFile(tt.args.filePath)
			cache.FileCache.SetFile(&protocol.TextDocumentItem{
				URI:  uri.File(tt.args.filePath),
				Text: string(content),
			})
			param := protocol.PublishDiagnosticsParams{URI: uri.File(tt.args.filePath)}
			got := Diagnostic(param, cache)
			if !reflect.DeepEqual(got.Diagnostics, tt.want) {
				t.Errorf("FindErrors() = %v, want %v", got.Diagnostics, tt.want)
			}
		})
	}
}
