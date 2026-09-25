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
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
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
	return completionLabelsWithCache(t, cache.New(), config, pos)
}

// completionLabelsWithCache are the labels completion offers at a position
// in a config at /config.yml, with what a cache remembers.
func completionLabelsWithCache(t *testing.T, c *cache.Cache, config string, pos protocol.Position) []string {
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
		Cache:   c,
		Context: settings,
	}
	ch.GetCompletionItems()

	labels := []string{}
	for _, item := range ch.Items {
		labels = append(labels, item.Label)
	}
	return labels
}

// positionBelow is the position on the line after the one whose text,
// trimmed, is the given text, at the given column.
func positionBelow(t *testing.T, config, text string, column uint32) protocol.Position {
	t.Helper()
	line := slices.IndexFunc(strings.Split(config, "\n"), func(l string) bool {
		return strings.TrimSpace(l) == text
	})
	assert.Assert(t, line != -1, "no line %q", text)
	return protocol.Position{Line: uint32(line + 1), Character: column}
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

func TestCompleteJobInvocationBody(t *testing.T) {
	const config = `version: 2.1

jobs:
  greet:
    parameters:
      who:
        type: string
      loud:
        type: boolean
        default: false
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo hi << parameters.who >>

workflows:
  main:
    jobs:
      - greet:
          who: me
          
      - hold:
          type: approval
          
      - greet:
          name: greet-again
          serial-group:
            
`
	below := func(text string, column uint32) protocol.Position {
		return positionBelow(t, config, text, column)
	}

	t.Run("a job's invocation is offered its keys and the parameters it isn't given", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(completionLabels(t, config, below("who: me", 10)), []string{
			"requires", "context", "filters", "matrix", "name", "type",
			"pre-steps", "post-steps", "serial-group", "override-with", "loud",
		}))
	})

	t.Run("an approval job is offered no parameters", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(completionLabels(t, config, below("type: approval", 10)), []string{
			"requires", "context", "filters", "matrix", "name",
			"pre-steps", "post-steps", "serial-group", "override-with",
		}))
	})

	t.Run("nothing is offered inside one of the invocation's keys", func(t *testing.T) {
		assert.Check(t, cmp.Len(completionLabels(t, config, below("serial-group:", 12)), 0))
	})
}

func TestCompleteInvocationMappings(t *testing.T) {
	const config = `version: 2.1

jobs:
  greet:
    parameters:
      who:
        type: string
      loud:
        type: boolean
        default: false
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo hi << parameters.who >>

workflows:
  main:
    jobs:
      - greet:
          filters:
            tags:
              only: /.*/
            
      - greet:
          name: greet-branches
          filters:
            branches:
              ignore: main
              
      - greet:
          name: greet-matrix
          matrix:
            alias: greet-all
            
      - greet:
          name: greet-each
          matrix:
            parameters:
              loud: [true, false]
              
`
	below := func(text string, column uint32) protocol.Position {
		return positionBelow(t, config, text, column)
	}

	t.Run("filters are offered the kinds of ref they don't filter yet", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(completionLabels(t, config, below("only: /.*/", 12)), []string{"branches"}))
	})

	t.Run("a kind of ref is offered the filters it doesn't have", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(completionLabels(t, config, below("ignore: main", 14)), []string{"only"}))
	})

	t.Run("a matrix is offered the keys it doesn't have", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(completionLabels(t, config, below("alias: greet-all", 12)), []string{"parameters", "exclude"}))
	})

	t.Run("a matrix's parameters are offered the job's parameters they don't have", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(completionLabels(t, config, below("loud: [true, false]", 14)), []string{"who"}))
	})
}

func TestCompleteRequiredStatus(t *testing.T) {
	const config = `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - checkout

workflows:
  main:
    jobs:
      - build
      - build:
          name: after-build
          requires:
            - build: fa
      - build:
          name: after-both
          requires:
            - build: [success, can]
`
	at := func(text string) protocol.Position {
		lines := strings.Split(config, "\n")
		line := slices.IndexFunc(lines, func(l string) bool { return strings.TrimSpace(l) == text })
		assert.Assert(t, line != -1, "no line %q", text)
		return protocol.Position{Line: uint32(line), Character: uint32(len(lines[line]))}
	}

	t.Run("a required job is offered the statuses it can be required to have", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(completionLabels(t, config, at("- build: fa")), []string{
			"success", "failed", "canceled", "not_run", "unauthorized", "terminal",
		}))
	})

	t.Run("a list of statuses is offered each status but terminal", func(t *testing.T) {
		inList := at("- build: [success, can]")
		inList.Character--
		assert.Check(t, cmp.DeepEqual(completionLabels(t, config, inList), []string{
			"success", "failed", "canceled", "not_run", "unauthorized",
		}))
	})
}

func TestCompleteContextName(t *testing.T) {
	const orgID = "11111111-2222-3333-4444-555555555555"
	const config = `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - checkout

workflows:
  main:
    jobs:
      - build:
          context: acme/
      - build:
          name: build-list
          context:
            - acme/build
            - ac
      - build:
          name: build-flow
          context: [acme/build, ac]
`
	fake := fakes.NewCircleCI(t)
	fake.AddContext(orgID, "ctx-deploy", "acme/deploy")
	fake.AddContext(orgID, "ctx-build", "acme/build")

	c := cache.New()
	t.Run("remember the organization's contexts", func(t *testing.T) {
		assert.NilError(t, c.LoadContexts(testHelpers.SettingsForHost(fake.URL()).Api, orgID))
		c.FileCache.SetFile(cache.File{TextDocument: protocol.TextDocumentItem{URI: uri.File("/config.yml")}})
		c.FileCache.AddProjectSlugToFile(uri.File("/config.yml"), circleci.Project{Slug: "gh/acme/rocket", OrganizationId: orgID})
	})

	lines := strings.Split(config, "\n")
	at := func(text string, fromEnd uint32) protocol.Position {
		line := slices.IndexFunc(lines, func(l string) bool { return strings.TrimSpace(l) == text })
		assert.Assert(t, line != -1, "no line %q", text)
		return protocol.Position{Line: uint32(line), Character: uint32(len(lines[line])) - fromEnd}
	}
	want := []string{"acme/build", "acme/deploy"}

	t.Run("a context is offered the organization's contexts", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(completionLabelsWithCache(t, c, config, at("context: acme/", 0)), want))
	})

	t.Run("so is an item of a list of contexts", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(completionLabelsWithCache(t, c, config, at("- ac", 0)), want))
	})

	t.Run("and of a flow list of them", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(completionLabelsWithCache(t, c, config, at("context: [acme/build, ac]", 1)), want))
	})

	t.Run("nothing is offered without the organization's contexts", func(t *testing.T) {
		assert.Check(t, cmp.Len(completionLabels(t, config, at("context: acme/", 0)), 0))
	})
}

func TestCompletePreAndPostSteps(t *testing.T) {
	const config = `version: 2.1

commands:
  greet:
    parameters:
      who:
        type: string
      loud:
        type: boolean
        default: false
    steps:
      - run: echo hi << parameters.who >>

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - checkout

workflows:
  main:
    jobs:
      - build:
          pre-steps:
            - run: echo before
            - 
          post-steps:
            - greet:
                who: me
                
`
	t.Run("pre-steps are offered steps", func(t *testing.T) {
		labels := completionLabels(t, config, positionBelow(t, config, "- run: echo before", 14))
		for _, want := range []string{"greet", "run", "checkout"} {
			assert.Check(t, cmp.Contains(labels, want))
		}
		assert.Check(t, !slices.Contains(labels, "build"), "a job offered as a step: %q", labels)
	})

	t.Run("a post-step is offered the parameters it isn't given", func(t *testing.T) {
		labels := completionLabels(t, config, positionBelow(t, config, "who: me", 16))
		assert.Check(t, cmp.DeepEqual(labels, []string{"loud"}))
	})
}
