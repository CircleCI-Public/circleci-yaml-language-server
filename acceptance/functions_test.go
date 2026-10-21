package acceptance

import (
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
)

// functionDeclarationConfig declares a function on line 3.
const functionDeclarationConfig = `version: 2.1

functions:
  setup-go: github.com/circleci-functions/setup-go@v0

jobs:
  build:
    machine:
      image: ubuntu-2404:current
    steps:
      - setup-go

workflows:
  main:
    jobs:
      - build
`

// TestFunctionDeclarationCompletion completes a declared function's name and
// version, each item replacing what is written.
func TestFunctionDeclarationCompletion(t *testing.T) {
	fake := linkedProjectFake(t)
	fake.AddFunction("fn-setup-go", "github.com/circleci-functions/setup-go", "Install a Go toolchain.",
		fakes.FunctionVersion{ID: "ver-1", Version: "v0.5.3-4edeb2a", Descriptor: map[string]any{"name": "setup-go"}},
		fakes.FunctionVersion{ID: "ver-2", Version: "v0.6.0-1a2b3c4", Descriptor: map[string]any{"name": "setup-go"}},
	)

	session := start(t, fake, functionDeclarationConfig, testToken)
	session.open(t, functionDeclarationConfig)

	edits := func(t *testing.T, pos protocol.Position) map[string]protocol.TextEdit {
		t.Helper()
		list, err := session.client.Completion(session.workspace.URI(), pos)
		assert.NilError(t, err)
		assert.Assert(t, list != nil, "no completion")
		got := map[string]protocol.TextEdit{}
		for _, item := range list.Items {
			edit, ok := item.TextEdit.(*protocol.TextEdit)
			if assert.Check(t, ok, "item %q has no text edit", item.Label) {
				got[item.Label] = *edit
			}
		}
		return got
	}

	t.Run("the name is completed to a published function at its latest version", func(t *testing.T) {
		end := position(3, uint32(len("  setup-go: github.com/circleci-functions/set")))
		reference := "github.com/circleci-functions/setup-go@v0.6.0-1a2b3c4"
		got := edits(t, end)
		assert.Check(t, cmp.DeepEqual(got, map[string]protocol.TextEdit{
			reference: {Range: protocol.Range{Start: position(3, uint32(len("  setup-go: "))), End: end}, NewText: reference},
		}))
	})

	t.Run("the version is completed to the function's versions", func(t *testing.T) {
		end := position(3, uint32(len("  setup-go: github.com/circleci-functions/setup-go@v0")))
		start := position(3, end.Character-uint32(len("v0")))
		got := edits(t, end)
		assert.Check(t, cmp.DeepEqual(got, map[string]protocol.TextEdit{
			"v0.5.3-4edeb2a": {Range: protocol.Range{Start: start, End: end}, NewText: "v0.5.3-4edeb2a"},
			"v0.6.0-1a2b3c4": {Range: protocol.Range{Start: start, End: end}, NewText: "v0.6.0-1a2b3c4"},
		}))
	})
}
