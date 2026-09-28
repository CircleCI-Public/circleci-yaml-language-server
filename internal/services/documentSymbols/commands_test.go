package documentSymbols

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestCommandParametersSymbolRange(t *testing.T) {
	doc := parseDoc(t, `version: 2.1
commands:
  greet:
    parameters:
      who:
        type: string
    steps:
      - run: echo << parameters.who >>
`)
	symbols := resolveCommandsSymbols(&doc)
	assert.Assert(t, cmp.Len(symbols, 1))
	assert.Assert(t, cmp.Len(symbols[0].Children, 1))

	var names []string
	for _, child := range symbols[0].Children[0].Children {
		names = append(names, child.Name)
		if child.Name == "Parameters" {
			assert.Check(t, cmp.Equal(child.Range.Start.Line, uint32(4)), "should start at `who:`, not at the steps")
		}
	}
	assert.Check(t, cmp.Contains(names, "Parameters"))
}
