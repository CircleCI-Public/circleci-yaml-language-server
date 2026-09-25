package complete

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestCompletePipelineParameters(t *testing.T) {
	const config = `version: 2.1

parameters:
  deploy:
    type: 
  env:
    type: enum
    
  target:
    type: string
    default: prod
    

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - checkout
`
	t.Run("a pipeline parameter's type is offered the pipeline's types", func(t *testing.T) {
		pos := positionBelow(t, config, "deploy:", 10)
		got := completionLabels(t, config, pos)
		assert.Check(t, cmp.DeepEqual(got, []string{"string", "boolean", "integer", "enum"}))
	})

	t.Run("an enum parameter without values is offered the enum key", func(t *testing.T) {
		pos := positionBelow(t, config, "type: enum", 4)
		got := completionLabels(t, config, pos)
		assert.Check(t, cmp.DeepEqual(got, []string{"default", "description", "enum"}))
	})

	t.Run("a parameter is offered the keys it doesn't have", func(t *testing.T) {
		pos := positionBelow(t, config, "default: prod", 4)
		got := completionLabels(t, config, pos)
		assert.Check(t, cmp.DeepEqual(got, []string{"description"}))
	})
}
