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

func TestExecutor(t *testing.T) {
	const config = `version: 2.1

orbs:
  tools:
    executors:
      tiny:
        description: The smallest there is.
        docker:
          - image: cimg/base:stable
    jobs:
      check:
        executor: tiny
        steps:
          - checkout

executors:
  node:
    description: Node, at a version.
    parameters:
      version:
        type: string
        default: "22.0"
    docker:
      - image: cimg/node:<< parameters.version >>

jobs:
  build:
    executor: node
    steps:
      - checkout
  test:
    executor:
      name: node
      version: "20.0"
    steps:
      - checkout
`
	doc, err := yamlparser.ParseFromContent([]byte(config), testHelpers.DefaultSettings(), uri.File("/config.yml"), protocol.Position{})
	assert.NilError(t, err)
	t.Cleanup(doc.Close)

	const node = "**node** executor\n\nNode, at a version.\n\nParameters:\n\n- `version` (string, default `22.0`)"

	t.Run("a job's executor shows its description and parameters", func(t *testing.T) {
		got, ok := Executor(doc, cache.New(), protocol.Position{Line: 27, Character: 16})
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(got, node))
	})

	t.Run("so does one given as a mapping, on its executor line", func(t *testing.T) {
		got, ok := Executor(doc, cache.New(), protocol.Position{Line: 31, Character: 6})
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(got, node))
	})

	t.Run("but not on its parameters", func(t *testing.T) {
		_, ok := Executor(doc, cache.New(), protocol.Position{Line: 33, Character: 8})
		assert.Check(t, !ok)
	})

	t.Run("an inline orb's job shows the orb's executor", func(t *testing.T) {
		got, ok := Executor(doc, cache.New(), protocol.Position{Line: 11, Character: 18})
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(got, "**tiny** executor\n\nThe smallest there is."))
	})
}
