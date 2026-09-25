package hover

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
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestStep(t *testing.T) {
	const config = `version: 2.1

orbs:
  tools:
    commands:
      lint:
        description: Lint the code.
        steps:
          - run: make lint
    jobs:
      check:
        docker:
          - image: cimg/base:stable
        steps:
          - lint

commands:
  greet:
    description: Say hello.
    parameters:
      who:
        type: string
        description: Who to greet.
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
      - greet:
          who: me
      - run: echo done
`
	doc, err := yamlparser.ParseFromContent([]byte(config), testHelpers.DefaultSettings(), uri.File("/config.yml"), protocol.Position{})
	assert.NilError(t, err)
	t.Cleanup(doc.Close)

	at := func(text string) protocol.Position {
		lines := strings.Split(config, "\n")
		line := slices.IndexFunc(lines, func(l string) bool { return strings.TrimSpace(l) == text })
		assert.Assert(t, line != -1, "no line %q", text)
		return protocol.Position{Line: uint32(line), Character: uint32(strings.Index(lines[line], "- ") + 3)}
	}

	t.Run("a command's step shows its description and parameters", func(t *testing.T) {
		got, ok := Step(doc, cache.New(), at("- greet:"))
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(got, "**greet** command\n\nSay hello.\n\nParameters:\n"+
			"\n- `loud` (boolean, default `false`)"+
			"\n- `who` (string, required): Who to greet."))
	})

	t.Run("an inline orb's step shows the orb's command", func(t *testing.T) {
		got, ok := Step(doc, cache.New(), at("- lint"))
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(got, "**lint** command\n\nLint the code."))
	})

	t.Run("a built-in step has none", func(t *testing.T) {
		_, ok := Step(doc, cache.New(), at("- run: echo done"))
		assert.Check(t, !ok)
	})
}
