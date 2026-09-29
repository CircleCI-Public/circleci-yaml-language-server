package complete

import (
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

// valuesConfig ends each value with a dot, as if one were being typed, and
// leaves a blank line where a job, a step and a parameter each take a key.
const valuesConfig = `version: 2.1

parameters:
  greeting:
    type: string
    default: hello.

executors:
  exec:
    docker:
      - image: cimg/base:stable
    working_directory: /tmp/exec.

commands:
  greet:
    parameters:
      who:
        type: string

    steps:
      - run:
          name: say.
          command: echo hi.

jobs:
  build:
    docker:
      - image: cimg/base:stable
    working_directory: ~/src.

    shell: /bin/bash -eo pipefail.
    environment:
      FOO: bar.
    steps:
      - checkout:
          path: ~/code.
      - restore_cache:
          key: v1-deps.
      - run:
          command: make test.

      - run:
          command: |
            make lint.
      - run: echo done.
    description: builds.

workflows:
  main:
    jobs:
      - build:
          context: org.
          name: build-it.
          filters:
            branches:
              only: main.
`

func TestCompleteNothingInValues(t *testing.T) {
	lines := strings.Split(valuesConfig, "\n")
	for i, line := range lines {
		if !strings.HasSuffix(line, ".") {
			continue
		}
		t.Run(strings.TrimSpace(line), func(t *testing.T) {
			got := completionLabels(t, valuesConfig, protocol.Position{Line: uint32(i), Character: uint32(len(line))})
			assert.Check(t, cmp.Len(got, 0))
		})
	}
}

func TestCompleteKeysBesideValues(t *testing.T) {
	blankBelow := func(text string, column uint32) protocol.Position {
		return positionBelow(t, valuesConfig, text, column)
	}

	t.Run("a job's keys", func(t *testing.T) {
		got := completionLabels(t, valuesConfig, blankBelow("working_directory: ~/src.", 4))
		assert.Check(t, cmp.Contains(got, "resource_class"))
	})

	t.Run("a step's keys", func(t *testing.T) {
		got := completionLabels(t, valuesConfig, blankBelow("command: make test.", 10))
		assert.Check(t, cmp.Contains(got, "working_directory"))
	})

	t.Run("a parameter's keys", func(t *testing.T) {
		got := completionLabels(t, valuesConfig, blankBelow("type: string", 8))
		assert.Check(t, cmp.Contains(got, "description"))
	})
}
