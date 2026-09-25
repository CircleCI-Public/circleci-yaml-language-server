package complete

import (
	"slices"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestCompleteInInlineOrb(t *testing.T) {
	const config = `version: 2.1

orbs:
  tools:
    executors:
      small:
        docker:
          - image: cimg/base:stable
    commands:
      greet:
        parameters:
          who:
            type: string
          loud:
            type: boolean
          count:
            type: 
        steps:
          - run: echo << parameters.who >>
    jobs:
      hello:
        executor: small
        steps:
          - 
          - greet:
              who: me
              

commands:
  outside:
    steps:
      - checkout

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - checkout

workflows:
  main:
    jobs:
      - tools/hello
`
	t.Run("a step is offered the orb's commands", func(t *testing.T) {
		pos := positionBelow(t, config, "executor: small", 10)
		pos.Line++
		got := completionLabels(t, config, pos)
		assert.Check(t, cmp.Contains(got, "greet"))
		assert.Check(t, cmp.Contains(got, "run"))
		assert.Check(t, !slices.Contains(got, "outside"), "the config's own command offered in the orb")
	})

	t.Run("an orb command's step is offered the parameters it isn't given", func(t *testing.T) {
		pos := positionBelow(t, config, "who: me", 14)
		got := completionLabels(t, config, pos)
		assert.Check(t, cmp.DeepEqual(got, []string{"count", "loud"}))
	})

	t.Run("an orb command's parameter type is offered the types", func(t *testing.T) {
		pos := positionBelow(t, config, "count:", 18)
		got := completionLabels(t, config, pos)
		assert.Check(t, cmp.DeepEqual(got, parameterTypes))
	})
}
