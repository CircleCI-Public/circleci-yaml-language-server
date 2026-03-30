package complete

import (
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"go.lsp.dev/protocol"
)

func TestFindJobGroup(t *testing.T) {
	doc := yamlparser.YamlDocument{
		JobGroups: map[string]ast.JobGroup{
			"frontend": {
				Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}, End: protocol.Position{Line: 5, Character: 0}},
				Name:  "frontend",
			},
			"backend": {
				Range: protocol.Range{Start: protocol.Position{Line: 7, Character: 0}, End: protocol.Position{Line: 10, Character: 0}},
				Name:  "backend",
			},
		},
	}

	tests := []struct {
		name     string
		pos      protocol.Position
		wantName string
		wantErr  bool
	}{
		{"match first job group", protocol.Position{Line: 3, Character: 0}, "frontend", false},
		{"match second job group", protocol.Position{Line: 8, Character: 0}, "backend", false},
		{"match start of range", protocol.Position{Line: 1, Character: 0}, "frontend", false},
		{"match end of range", protocol.Position{Line: 5, Character: 0}, "frontend", false},
		{"no match - between groups", protocol.Position{Line: 6, Character: 0}, "", true},
		{"no match - before all", protocol.Position{Line: 0, Character: 0}, "", true},
		{"no match - after all", protocol.Position{Line: 11, Character: 0}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jg, err := findJobGroup(tt.pos, doc)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got job group %q", jg.Name)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if jg.Name != tt.wantName {
					t.Errorf("expected name %q, got %q", tt.wantName, jg.Name)
				}
			}
		})
	}
}

func TestFindJobGroupEmptyDoc(t *testing.T) {
	doc := yamlparser.YamlDocument{}
	_, err := findJobGroup(protocol.Position{Line: 0, Character: 0}, doc)
	if err == nil {
		t.Error("expected error for empty document")
	}
}

func TestAddJobGroupsCompletion(t *testing.T) {
	tests := []struct {
		name       string
		jobGroups  map[string]ast.JobGroup
		wantLabels []string
	}{
		{
			name: "adds all job group names",
			jobGroups: map[string]ast.JobGroup{
				"frontend": {Name: "frontend"},
				"backend":  {Name: "backend"},
			},
			wantLabels: []string{"frontend", "backend"},
		},
		{
			name:       "empty job groups",
			jobGroups:  map[string]ast.JobGroup{},
			wantLabels: nil,
		},
		{
			name: "single job group",
			jobGroups: map[string]ast.JobGroup{
				"only-group": {Name: "only-group"},
			},
			wantLabels: []string{"only-group"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := &CompletionHandler{
				Doc: yamlparser.YamlDocument{
					JobGroups: tt.jobGroups,
				},
			}
			ch.addJobGroupsCompletion()

			if len(ch.Items) != len(tt.wantLabels) {
				t.Fatalf("got %d items, want %d", len(ch.Items), len(tt.wantLabels))
			}

			// Since map iteration order is non-deterministic, check labels as a set
			gotLabels := make(map[string]bool)
			for _, item := range ch.Items {
				gotLabels[item.Label] = true
			}
			for _, want := range tt.wantLabels {
				if !gotLabels[want] {
					t.Errorf("missing expected label %q", want)
				}
			}
		})
	}
}
