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
