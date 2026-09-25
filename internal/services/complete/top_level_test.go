package complete

import (
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestCompleteTopLevel(t *testing.T) {
	const config = `version: 

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - checkout

`
	t.Run("the version is offered the config format's", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(completionLabels(t, config, protocol.Position{Line: 0, Character: 9}), []string{"2.1"}))
	})

	t.Run("a top-level key is offered the keys the config doesn't have", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(completionLabels(t, config, positionBelow(t, config, "- checkout", 0)), []string{
			"setup", "orbs", "functions", "parameters", "executors", "commands", "job-groups", "workflows",
		}))
	})
}
