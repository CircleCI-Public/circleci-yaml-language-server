package complete

import (
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestCompleteFunctionVersion(t *testing.T) {
	const config = `version: 2.1

functions:
  setup-go: github.com/circleci-functions/setup-go@v0
  unknown: github.com/circleci-functions/unknown@

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - setup-go
`
	fake := fakes.NewCircleCI(t)
	fake.AddFunction("fn-setup-go", "github.com/circleci-functions/setup-go", "Install a Go toolchain.",
		fakes.FunctionVersion{ID: "ver-1", Version: "v0.5.1-684fd5b", Descriptor: map[string]any{"name": "setup-go"}},
		fakes.FunctionVersion{ID: "ver-2", Version: "v0.6.0-1a2b3c4", Descriptor: map[string]any{"name": "setup-go"}},
	)
	settings := testHelpers.SettingsForHost(fake.URL())

	t.Run("a function's version is offered its published versions", func(t *testing.T) {
		pos := protocol.Position{Line: 3, Character: uint32(len("  setup-go: github.com/circleci-functions/setup-go@v0"))}
		got := completionLabelsWith(t, settings, cache.New(), config, pos)
		assert.Check(t, cmp.DeepEqual(got, []string{
			"v0.5.1-684fd5b", "v0.6.0-1a2b3c4",
		}))
	})

	t.Run("an unpublished function's is offered none", func(t *testing.T) {
		pos := protocol.Position{Line: 4, Character: uint32(len("  unknown: github.com/circleci-functions/unknown@"))}
		got := completionLabelsWith(t, settings, cache.New(), config, pos)
		assert.Check(t, cmp.Len(got, 0))
	})
}
