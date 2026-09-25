package hover

import (
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestOrbDeclaration(t *testing.T) {
	const config = `version: 2.1

orbs:
  tools:
    description: Our own tools.
    commands:
      lint:
        steps:
          - run: make lint
    executors:
      tiny:
        docker:
          - image: cimg/base:stable

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - tools/lint
`
	doc, err := yamlparser.ParseFromContent([]byte(config), testHelpers.DefaultSettings(), uri.File("/config.yml"), protocol.Position{})
	assert.NilError(t, err)
	t.Cleanup(doc.Close)

	t.Run("an inline orb's name shows the orb", func(t *testing.T) {
		got, ok := OrbDeclaration(doc, cache.New(), protocol.Position{Line: 3, Character: 4})
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(got, "**tools** orb\n\nOur own tools.\n\nCommands: `lint`\n\nExecutors: `tiny`"))
	})

	t.Run("its body doesn't", func(t *testing.T) {
		_, ok := OrbDeclaration(doc, cache.New(), protocol.Position{Line: 6, Character: 8})
		assert.Check(t, !ok)
	})
}
