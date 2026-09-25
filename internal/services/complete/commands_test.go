package complete

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestCompleteCommandKeys(t *testing.T) {
	const config = `version: 2.1

commands:
  greet:
    steps:
      - run: echo hi
    
  wave:
    description: Wave.
    parameters:
      who:
        type: string
    steps:
      - run: echo bye << parameters.who >>
    
`
	t.Run("a command is offered the keys it doesn't have", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(completionLabels(t, config, positionBelow(t, config, "- run: echo hi", 4)), []string{
			"description", "parameters",
		}))
	})

	t.Run("a command with every key is offered none", func(t *testing.T) {
		assert.Check(t, cmp.Len(completionLabels(t, config, positionBelow(t, config, "- run: echo bye << parameters.who >>", 4)), 0))
	})
}
