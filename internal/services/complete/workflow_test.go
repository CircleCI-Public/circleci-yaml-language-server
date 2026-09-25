package complete

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
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

// completionLabels are the labels completion offers at a position in a config.
func completionLabels(t *testing.T, config string, pos protocol.Position) []string {
	t.Helper()

	settings := testHelpers.DefaultSettings()
	doc, err := yamlparser.ParseFromContent([]byte(config), settings, uri.File("/config.yml"), protocol.Position{})
	assert.NilError(t, err)
	t.Cleanup(doc.Close)

	ch := CompletionHandler{
		Params: protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{Position: pos},
		},
		Doc:     doc,
		Cache:   cache.New(),
		Context: settings,
	}
	ch.GetCompletionItems()

	labels := []string{}
	for _, item := range ch.Items {
		labels = append(labels, item.Label)
	}
	return labels
}

func TestCompleteWorkflowKeys(t *testing.T) {
	const config = `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - checkout

workflows:
  main:
    when: << pipeline.git.branch >>
    jobs:
      - build:
          context: 
    
`

	t.Run("a workflow's missing keys are offered at its keys", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(
			completionLabels(t, config, protocol.Position{Line: 15, Character: 4}),
			[]string{"triggers", "unless", "max_auto_reruns"},
		))
	})

	t.Run("nothing of the workflow's is offered inside a job invocation", func(t *testing.T) {
		labels := completionLabels(t, config, protocol.Position{Line: 14, Character: 19})
		assert.Check(t, !slices.Contains(labels, "triggers"), "%q", labels)
		assert.Check(t, !slices.Contains(labels, "trigger"), "%q", labels)
	})
}

func TestCompleteJobKeys(t *testing.T) {
	jobKeys := func(t *testing.T, body string) []string {
		t.Helper()
		// The cursor is on the job's last line, which is left blank.
		config := "version: 2.1\n\njobs:\n  j:\n" + body + "    \n"
		line := uint32(4 + strings.Count(body, "\n"))
		return completionLabels(t, config, protocol.Position{Line: line, Character: 4})
	}
	anyOrder := cmpopts.SortSlices(func(a, b string) bool { return a < b })

	t.Run("a job with no executor is offered each kind", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(jobKeys(t, "    steps:\n      - checkout\n"), []string{
			"description", "executor", "docker", "machine", "macos", "resource_class", "shell",
			"working_directory", "environment", "parameters", "parallelism", "circleci_ip_ranges",
			"retention", "type",
		}, anyOrder))
	})

	t.Run("a job with an executor isn't offered another", func(t *testing.T) {
		labels := jobKeys(t, "    docker:\n      - image: cimg/base:stable\n")
		for _, key := range []string{"executor", "docker", "machine", "macos"} {
			assert.Check(t, !slices.Contains(labels, key), "%s offered: %q", key, labels)
		}
		assert.Check(t, cmp.Contains(labels, "parameters"))
		assert.Check(t, cmp.Contains(labels, "parallelism"))
	})

	t.Run("a release job is offered its plan", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(jobKeys(t, "    type: release\n"), []string{"plan_name"}))
	})

	t.Run("a lock job is offered its key and parameters", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(jobKeys(t, "    type: lock\n"), []string{"key", "parameters"}, anyOrder))
	})

	t.Run("an approval job is offered nothing", func(t *testing.T) {
		assert.Check(t, cmp.Len(jobKeys(t, "    type: approval\n"), 0))
	})
}
