package complete

import (
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"go.lsp.dev/protocol"
)

func TestFindWorkflow(t *testing.T) {
	doc := yamlparser.YamlDocument{
		Workflows: map[string]ast.Workflow{
			"build": {
				Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}, End: protocol.Position{Line: 5, Character: 0}},
				Name:  "build",
			},
			"deploy": {
				Range: protocol.Range{Start: protocol.Position{Line: 7, Character: 0}, End: protocol.Position{Line: 10, Character: 0}},
				Name:  "deploy",
			},
		},
	}

	tests := []struct {
		name     string
		pos      protocol.Position
		wantName string
		wantErr  bool
	}{
		{"match first workflow", protocol.Position{Line: 3, Character: 0}, "build", false},
		{"match second workflow", protocol.Position{Line: 8, Character: 0}, "deploy", false},
		{"match start of range", protocol.Position{Line: 1, Character: 0}, "build", false},
		{"match end of range", protocol.Position{Line: 5, Character: 0}, "build", false},
		{"no match - between workflows", protocol.Position{Line: 6, Character: 0}, "", true},
		{"no match - before all", protocol.Position{Line: 0, Character: 0}, "", true},
		{"no match - after all", protocol.Position{Line: 11, Character: 0}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf, err := findWorkflow(tt.pos, doc)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got workflow %q", wf.Name)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if wf.Name != tt.wantName {
					t.Errorf("expected name %q, got %q", tt.wantName, wf.Name)
				}
			}
		})
	}
}

func TestFindWorkflowEmptyDoc(t *testing.T) {
	doc := yamlparser.YamlDocument{}
	_, err := findWorkflow(protocol.Position{Line: 0, Character: 0}, doc)
	if err == nil {
		t.Error("expected error for empty document")
	}
}
