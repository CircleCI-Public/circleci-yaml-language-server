package complete

import (
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"go.lsp.dev/protocol"
)

func TestFindJobInvocation(t *testing.T) {
	invocations := []ast.JobInvocation{
		{JobName: "build", JobNameRange: protocol.Range{Start: protocol.Position{Line: 1, Character: 0}, End: protocol.Position{Line: 1, Character: 5}}},
		{JobName: "test", JobNameRange: protocol.Range{Start: protocol.Position{Line: 3, Character: 0}, End: protocol.Position{Line: 3, Character: 4}}},
		{JobName: "deploy", JobNameRange: protocol.Range{Start: protocol.Position{Line: 5, Character: 0}, End: protocol.Position{Line: 5, Character: 6}}},
	}

	tests := []struct {
		name        string
		pos         protocol.Position
		invocations []ast.JobInvocation
		wantName    string
		wantNil     bool
	}{
		{"match first invocation", protocol.Position{Line: 1, Character: 2}, invocations, "build", false},
		{"match start of range", protocol.Position{Line: 1, Character: 0}, invocations, "build", false},
		{"match end of range", protocol.Position{Line: 1, Character: 5}, invocations, "build", false},
		{"match second invocation", protocol.Position{Line: 3, Character: 2}, invocations, "test", false},
		{"match third invocation", protocol.Position{Line: 5, Character: 3}, invocations, "deploy", false},
		{"no match - wrong line", protocol.Position{Line: 2, Character: 0}, invocations, "", true},
		{"no match - before all", protocol.Position{Line: 0, Character: 0}, invocations, "", true},
		{"no match - after all", protocol.Position{Line: 6, Character: 0}, invocations, "", true},
		{"empty invocations", protocol.Position{Line: 1, Character: 0}, nil, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findJobInvocation(tt.pos, tt.invocations)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
			} else {
				if result == nil {
					t.Fatal("expected non-nil result")
				}
				if result.JobName != tt.wantName {
					t.Errorf("expected JobName %q, got %q", tt.wantName, result.JobName)
				}
			}
		})
	}
}

func TestIsInRequires(t *testing.T) {
	invocations := []ast.JobInvocation{
		{
			JobName: "build",
			Requires: []ast.Require{
				{Name: "setup", Range: protocol.Range{Start: protocol.Position{Line: 10, Character: 4}, End: protocol.Position{Line: 10, Character: 9}}},
				{Name: "lint", Range: protocol.Range{Start: protocol.Position{Line: 11, Character: 4}, End: protocol.Position{Line: 11, Character: 8}}},
			},
		},
		{
			JobName:  "test",
			Requires: []ast.Require{},
		},
		{
			JobName: "deploy",
			Requires: []ast.Require{
				{Name: "build", Range: protocol.Range{Start: protocol.Position{Line: 20, Character: 4}, End: protocol.Position{Line: 20, Character: 9}}},
			},
		},
	}

	tests := []struct {
		name        string
		pos         protocol.Position
		invocations []ast.JobInvocation
		want        bool
	}{
		{"in first require of first job", protocol.Position{Line: 10, Character: 6}, invocations, true},
		{"in second require of first job", protocol.Position{Line: 11, Character: 5}, invocations, true},
		{"in require of third job", protocol.Position{Line: 20, Character: 4}, invocations, true},
		{"not in any require - wrong line", protocol.Position{Line: 12, Character: 0}, invocations, false},
		{"not in any require - before all", protocol.Position{Line: 9, Character: 0}, invocations, false},
		{"empty invocations", protocol.Position{Line: 10, Character: 6}, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isInRequires(tt.pos, tt.invocations)
			if got != tt.want {
				t.Errorf("isInRequires() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddExistingJobInvocations(t *testing.T) {
	tests := []struct {
		name        string
		invocations []ast.JobInvocation
		wantLabels  []string
	}{
		{
			name: "adds step names as completion items",
			invocations: []ast.JobInvocation{
				{StepName: "build-step"},
				{StepName: "test-step"},
				{StepName: "deploy-step"},
			},
			wantLabels: []string{"build-step", "test-step", "deploy-step"},
		},
		{
			name:        "empty invocations adds nothing",
			invocations: []ast.JobInvocation{},
			wantLabels:  nil,
		},
		{
			name: "single invocation",
			invocations: []ast.JobInvocation{
				{StepName: "only-job"},
			},
			wantLabels: []string{"only-job"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := &CompletionHandler{}
			ch.addExistingJobInvocations(tt.invocations)

			if len(ch.Items) != len(tt.wantLabels) {
				t.Fatalf("got %d items, want %d", len(ch.Items), len(tt.wantLabels))
			}
			for i, want := range tt.wantLabels {
				if ch.Items[i].Label != want {
					t.Errorf("item[%d].Label = %q, want %q", i, ch.Items[i].Label, want)
				}
			}
		})
	}
}
