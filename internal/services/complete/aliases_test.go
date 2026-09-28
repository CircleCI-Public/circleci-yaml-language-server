package complete

import (
	"slices"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

const aliasesConfig = `version: 2.1

orbs:
  orb:
    executors:
      e:
        machine:
          image: ubuntu-2404:current
    commands:
      c:
        steps:
          - run: echo hello
    jobs:
      build:
        machine:
          image: ubuntu-2404:current
        steps:
          - c

executors:
  my-exec: orb/e

commands:
  renamed-c: orb/c

jobs:
  renamed-build: orb/build
  local:
    executor: my
    steps:
      - 

workflows:
  main:
    jobs:
      - 
`

func TestCompleteAliases(t *testing.T) {
	lines := strings.Split(aliasesConfig, "\n")
	lineOf := func(t *testing.T, text string, from int) uint32 {
		t.Helper()
		i := slices.Index(lines[from:], text)
		assert.Assert(t, i != -1, "no line %q", text)
		return uint32(from + i)
	}

	t.Run("a step is offered a command's alias", func(t *testing.T) {
		step := lineOf(t, "      - ", 0)
		labels := completionLabels(t, aliasesConfig, protocol.Position{Line: step, Character: 8})
		assert.Check(t, cmp.Contains(labels, "renamed-c"))
	})

	t.Run("a workflow is offered a job's alias", func(t *testing.T) {
		invocation := lineOf(t, "      - ", int(lineOf(t, "workflows:", 0)))
		labels := completionLabels(t, aliasesConfig, protocol.Position{Line: invocation, Character: 8})
		assert.Check(t, cmp.Contains(labels, "renamed-build"))
	})

	t.Run("a job's executor is offered an executor's alias", func(t *testing.T) {
		executor := lineOf(t, "    executor: my", 0)
		labels := completionLabels(t, aliasesConfig, protocol.Position{Line: executor, Character: 16})
		assert.Check(t, cmp.Contains(labels, "my-exec"))
	})
}
