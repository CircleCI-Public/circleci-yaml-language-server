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

func TestAliases(t *testing.T) {
	const config = `version: 2.1

orbs:
  orb:
    executors:
      e:
        description: The orb's machine.
        machine:
          image: ubuntu-2404:current
    commands:
      c:
        description: Say hello.
        steps:
          - run: echo hello
    jobs:
      build:
        description: Build it.
        executor: e
        steps:
          - c

executors:
  my-exec: orb/e

commands:
  renamed-c: orb/c

jobs:
  renamed-build: orb/build
  local:
    executor: my-exec
    steps:
      - renamed-c

workflows:
  main:
    jobs:
      - renamed-build
      - local
`
	doc, err := yamlparser.ParseFromContent(t.Context(), []byte(config), testHelpers.DefaultSettings(), uri.File("/config.yml"), protocol.Position{})
	assert.NilError(t, err)
	t.Cleanup(doc.Close)

	lines := strings.Split(config, "\n")
	at := func(t *testing.T, line string) protocol.Position {
		t.Helper()
		i := slices.Index(lines, line)
		assert.Assert(t, i != -1, "no line %q", line)
		return protocol.Position{Line: uint32(i), Character: uint32(len(line) - 2)}
	}

	t.Run("a step shows the command its alias names", func(t *testing.T) {
		got, ok := Step(doc, cache.New(), at(t, "      - renamed-c"))
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(got, "**renamed-c** command\n\nSay hello."))
	})

	t.Run("a workflow's job shows the job its alias names", func(t *testing.T) {
		got, ok := JobInvocation(doc, cache.New(), at(t, "      - renamed-build"))
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(got, "**renamed-build** job\n\nBuild it."))
	})

	t.Run("a job's executor shows the executor its alias names", func(t *testing.T) {
		got, ok := Executor(doc, cache.New(), at(t, "    executor: my-exec"))
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(got, "**my-exec** executor\n\nThe orb's machine."))
	})
}
