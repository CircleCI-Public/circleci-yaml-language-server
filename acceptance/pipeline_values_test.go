package acceptance

import (
	"slices"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

// pipelineValuesConfig uses one pipeline value, and leaves another half
// written.
const pipelineValuesConfig = `version: 2.1

jobs:
  build:
    machine:
      image: ubuntu-2404:current
    steps:
      - run: echo << pipeline.git.branch >>
      - run: echo << pipeline.git.

workflows:
  main:
    jobs:
      - build
`

func TestPipelineValues(t *testing.T) {
	fake := linkedProjectFake(t)
	session := start(t, fake, pipelineValuesConfig, testToken)
	session.open(t, pipelineValuesConfig)

	lines := strings.Split(pipelineValuesConfig, "\n")

	t.Run("hover shows what a value holds", func(t *testing.T) {
		const used = "      - run: echo << pipeline.git.branch >>"
		line := slices.Index(lines, used)
		assert.Assert(t, line >= 0)

		hover, err := session.client.Hover(session.workspace.URI(), position(uint32(line), uint32(strings.Index(used, "branch"))))
		assert.NilError(t, err)
		assert.Assert(t, hover != nil, "no hover")
		markup, ok := hover.Contents.(*protocol.MarkupContent)
		assert.Assert(t, ok, "hover contents are %T, not markup", hover.Contents)
		assert.Check(t, cmp.Contains(markup.Value, "**pipeline.git.branch** `string`"))
	})

	t.Run("completion offers the next part of a value", func(t *testing.T) {
		const partial = "      - run: echo << pipeline.git."
		line := slices.Index(lines, partial)
		assert.Assert(t, line >= 0)

		completions, err := session.client.Completion(session.workspace.URI(), position(uint32(line), uint32(len(partial))))
		assert.NilError(t, err)
		assert.Assert(t, completions != nil)

		labels := make([]string, 0, len(completions.Items))
		for _, item := range completions.Items {
			labels = append(labels, item.Label)
		}
		assert.Check(t, cmp.Contains(labels, "branch"))
		assert.Check(t, cmp.Contains(labels, "revision"))
	})
}
