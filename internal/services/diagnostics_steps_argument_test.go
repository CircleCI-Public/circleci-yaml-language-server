package languageservice

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
)

func TestDiagnosticsConditionalInStepsArgument(t *testing.T) {
	fake := fakes.NewCircleCI(t)
	unknown := []string{"Cannot find a definition for command named not-a-command"}

	t.Run("on a command step", func(t *testing.T) {
		for _, kind := range []string{"when", "unless"} {
			t.Run(kind, func(t *testing.T) {
				errors := configErrors(t, fake, `version: 2.1

commands:
  wrap:
    parameters:
      steps:
        type: steps
    steps:
      - steps: << parameters.steps >>

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - wrap:
          steps:
            - `+kind+`:
                condition: true
                steps:
                  - not-a-command

workflows:
  main:
    jobs:
      - build
`)
				assert.Check(t, cmp.DeepEqual(errors, unknown))
			})
		}
	})

	t.Run("written as a flow mapping on a command step", func(t *testing.T) {
		errors := configErrors(t, fake, `version: 2.1

commands:
  wrap:
    parameters:
      steps:
        type: steps
    steps:
      - steps: << parameters.steps >>

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - wrap:
          steps:
            - {run: echo}
            - {when: {condition: true, steps: [not-a-command]}}
            - checkout

workflows:
  main:
    jobs:
      - build
`)
		assert.Check(t, cmp.DeepEqual(errors, unknown))
	})

	t.Run("on a workflow job invocation", func(t *testing.T) {
		for _, kind := range []string{"when", "unless"} {
			t.Run(kind, func(t *testing.T) {
				errors := configErrors(t, fake, `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:current
    parameters:
      steps:
        type: steps
    steps:
      - steps: << parameters.steps >>

workflows:
  main:
    jobs:
      - build:
          steps:
            - `+kind+`:
                condition: true
                steps:
                  - not-a-command
`)
				assert.Check(t, cmp.DeepEqual(errors, unknown))
			})
		}
	})
}
