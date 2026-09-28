package validate

import (
	"slices"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestCheckNames(t *testing.T) {
	clash := func(diags []string) []string {
		var clashes []string
		for _, message := range diags {
			if strings.Contains(message, "is already used to define") {
				clashes = append(clashes, message)
			}
		}
		slices.Sort(clashes)
		return clashes
	}

	t.Run("a workflow named like its job", func(t *testing.T) {
		diags := diagnosticMessages(validateYAML(t, `version: 2.1
setup: true
jobs:
  setup:
    docker:
      - image: cimg/base:current
    steps:
      - checkout
workflows:
  setup:
    jobs:
      - setup
`))
		assert.Check(t, cmp.Len(clash(diags), 0))
	})

	t.Run("a command named like a job", func(t *testing.T) {
		diags := diagnosticMessages(validateYAML(t, `version: 2.1
commands:
  deploy:
    steps:
      - checkout
jobs:
  deploy:
    docker:
      - image: cimg/base:current
    steps:
      - deploy
workflows:
  main:
    jobs:
      - deploy
`))
		assert.Check(t, cmp.DeepEqual(clash(diags), []string{
			`The name "deploy" is already used to define a command. You might want to use a different name to avoid confusion.`,
			`The name "deploy" is already used to define a job. You might want to use a different name to avoid confusion.`,
		}))
	})
}
