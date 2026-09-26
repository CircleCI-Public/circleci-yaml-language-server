package complete

import (
	"slices"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

// stepsConfig has a job, a command and a declared function, and a job with an
// empty step, where the cursor goes.
const stepsConfig = `version: 2.1

functions:
  setup-go: github.com/circleci-functions/setup-go@v0.5.1-684fd5b

commands:
  greet:
    steps:
      - run: echo hello

jobs:
  build:
    steps:
      - 
  other:
    steps:
      - greet
`

func TestCompleteSteps(t *testing.T) {
	fake := fakes.NewCircleCI(t)
	fake.AddFunction("fn-setup-go", "github.com/circleci-functions/setup-go", "Install a Go toolchain.",
		fakes.FunctionVersion{ID: "ver-setup-go", Version: "v0.5.1-684fd5b", Descriptor: map[string]any{
			"name":     "setup-go",
			"commands": map[string]any{"cache": map[string]any{"description": "Restore and save the Go caches."}},
		}},
	)
	settings := testHelpers.SettingsForHost(fake.URL())
	emptyStep := uint32(slices.Index(strings.Split(stepsConfig, "\n"), "      - "))

	doc, err := yamlparser.ParseFromContent([]byte(stepsConfig), settings, uri.File("/config.yml"), protocol.Position{})
	assert.NilError(t, err)
	t.Cleanup(doc.Close)

	ch := CompletionHandler{
		Params: protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				Position: protocol.Position{Line: emptyStep, Character: 8},
			},
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

	t.Run("commands and declared functions are offered", func(t *testing.T) {
		for _, want := range []string{"greet", "run", "setup-go", "setup-go/cache"} {
			assert.Check(t, cmp.Contains(labels, want))
		}
	})

	t.Run("jobs are not", func(t *testing.T) {
		for _, job := range []string{"build", "other"} {
			assert.Check(t, !slices.Contains(labels, job), "job %q offered as a step", job)
		}
	})
}

func TestCompleteStepBody(t *testing.T) {
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
      - run:
          command: echo
          
      - greet:
          who: me
          
      - when:
          condition: true
          
      - run:
          environment:
            
`
	at := func(line uint32) []string {
		return completionLabels(t, config, protocol.Position{Line: line, Character: 10})
	}

	t.Run("a built-in step is offered the keys it doesn't have", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(at(20), []string{
			"name", "shell", "environment", "background", "working_directory",
			"no_output_timeout", "when", "max_auto_reruns", "auto_rerun_delay", "teardown",
		}))
	})

	t.Run("a command step is offered the parameters it isn't given", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(at(23), []string{"loud"}))
	})

	t.Run("a when step is offered its steps", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(at(26), []string{"steps"}))
	})

	t.Run("nothing is offered in a value of a step's body", func(t *testing.T) {
		assert.Check(t, cmp.Len(completionLabels(t, config, protocol.Position{Line: 29, Character: 12}), 0))
	})
}

func TestCompleteTeardown(t *testing.T) {
	withTeardown := func(teardown string) string {
		return `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          command: make test
          teardown:
` + teardown + `
`
	}

	t.Run("only the steps a teardown can hold are offered", func(t *testing.T) {
		config := withTeardown("            - ")
		labels := completionLabels(t, config, positionBelow(t, config, "teardown:", 14))
		assert.Check(t, cmp.DeepEqual(labels, teardownSteps))
	})

	t.Run("a teardown run can't run in the background or have a teardown", func(t *testing.T) {
		config := withTeardown("            - run:\n                ")
		lastLine := uint32(strings.Count(config, "\n") - 1)
		labels := completionLabels(t, config, protocol.Position{Line: lastLine, Character: 16})
		assert.Check(t, cmp.Contains(labels, "command"))
		assert.Check(t, !slices.Contains(labels, "background"), "%q", labels)
		assert.Check(t, !slices.Contains(labels, "teardown"), "%q", labels)
	})
}

func TestCompleteNoStepsInAStepsValue(t *testing.T) {
	for _, line := range []string{"      - run: echo ", "      - run: ", "      - greet: "} {
		t.Run(strings.TrimSpace(line), func(t *testing.T) {
			config := `version: 2.1

commands:
  greet:
    steps:
      - run: echo hello

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
` + line + `
`
			lastLine := uint32(strings.Count(config, "\n") - 1)
			labels := completionLabels(t, config, protocol.Position{Line: lastLine, Character: uint32(len(line))})
			assert.Check(t, cmp.Len(labels, 0))
		})
	}
}

func TestCompleteBuiltInStepValues(t *testing.T) {
	tests := []struct {
		step, key string
		want      []string
	}{
		{"run", "when", []string{"always", "on_success", "on_fail"}},
		{"save_cache", "when", []string{"always", "on_success", "on_fail"}},
		{"run", "background", []string{"true", "false"}},
		{"setup_remote_docker", "version", []string{"default", "24.0.9"}},
		{"setup_remote_docker", "prefer_same_region", []string{"true", "false"}},
		{"with_tool_cache", "tool", []string{"gradle", "bazel", "turborepo", "xcode"}},
		{"run", "command", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.step+"."+tt.key, func(t *testing.T) {
			config := `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - ` + tt.step + `:
          ` + tt.key + `: 
`
			pos := positionBelow(t, config, "- "+tt.step+":", uint32(len("          "+tt.key+": ")))
			assert.Check(t, cmp.DeepEqual(completionLabels(t, config, pos), tt.want))
		})
	}
}
