package parser

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestFindDeepestNode(t *testing.T) {
	content := []byte(`workflows:
  main:
    jobs:
      - build
jobs:
  build:
    steps:
      - checkout
      - run: make
  test:
    context: [one, 42, true]
    environment: {"A": 1, B: [x, {c: 2}]}
`)
	root := rootNodeOf(t, content)

	find := func(t *testing.T, path string) string {
		t.Helper()
		node, err := FindDeepestNode(root, content, strings.Split(path, "."))
		assert.NilError(t, err)
		return strings.TrimSpace(string(content[node.StartByte():node.EndByte()]))
	}

	t.Run("a top-level key, not the same key further down that comes first", func(t *testing.T) {
		assert.Check(t, cmp.Equal(find(t, "jobs.build.steps.1"), "- run: make"))
	})

	t.Run("a key below a sequence item", func(t *testing.T) {
		assert.Check(t, cmp.Equal(find(t, "workflows.main.jobs.0"), "- build"))
	})

	t.Run("an item of a flow sequence, past the brackets and commas", func(t *testing.T) {
		assert.Check(t, cmp.Equal(find(t, "jobs.test.context.2"), "true"))
	})

	t.Run("a pair of a flow mapping, quoted or not", func(t *testing.T) {
		assert.Check(t, cmp.Equal(find(t, "jobs.test.environment.A"), `"A": 1`))
		assert.Check(t, cmp.Equal(find(t, "jobs.test.environment.B.1.c"), "c: 2"))
	})
}
