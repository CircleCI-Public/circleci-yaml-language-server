package hover

import (
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

func TestFunctions(t *testing.T) {
	const config = `version: 2.1

functions:
  setup-go: github.com/circleci-functions/setup-go@v0.5.1

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - setup-go:
          with:
            version: "1.25"
      - setup-go/cache
`
	fake := fakes.NewCircleCI(t)
	fake.AddFunction("fn-setup-go", "github.com/circleci-functions/setup-go", "Install a Go toolchain.",
		fakes.FunctionVersion{ID: "ver-setup-go", Version: "v0.5.1", Descriptor: map[string]any{
			"name":        "setup-go",
			"description": "Install a Go toolchain.",
			"flags": []any{
				map[string]any{"name": "version", "type": "string", "default": "stable", "description": "The Go version."},
			},
			"commands": map[string]any{"cache": map[string]any{"description": "Restore and save the Go caches."}},
		}},
	)
	settings := testHelpers.SettingsForHost(fake.URL())

	doc, err := yamlparser.ParseFromContent([]byte(config), settings, uri.File("/config.yml"), protocol.Position{})
	assert.NilError(t, err)
	t.Cleanup(doc.Close)
	c := cache.New()

	const setupGo = "**setup-go** function\n\nInstall a Go toolchain.\n\n" +
		"Flags, under `with`:\n\n- `version` (string, default `stable`): The Go version."

	t.Run("a function's step shows its description and flags", func(t *testing.T) {
		got, ok := FunctionStep(doc, c, protocol.Position{Line: 10, Character: 10})
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(got, setupGo))
	})

	t.Run("a function command's step shows the command", func(t *testing.T) {
		got, ok := FunctionStep(doc, c, protocol.Position{Line: 13, Character: 10})
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(got, "**setup-go/cache** function command\n\nRestore and save the Go caches."))
	})

	t.Run("a declaration shows the function it declares", func(t *testing.T) {
		got, ok := FunctionDeclaration(doc, c, protocol.Position{Line: 3, Character: 4})
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(got, setupGo))
	})
}
