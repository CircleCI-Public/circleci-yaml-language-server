package complete

import (
	"slices"
	"strings"
	"testing"

	"go.lsp.dev/protocol"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestCompleteParameterValues(t *testing.T) {
	const config = `version: 2.1

executors:
  base:
    parameters:
      size:
        type: enum
        enum: [small, medium, large]
        default: medium
    docker:
      - image: cimg/base:stable
    resource_class: << parameters.size >>

commands:
  greet:
    parameters:
      loud:
        type: boolean
        default: false
    steps:
      - run: echo hi

jobs:
  build:
    parameters:
      os:
        type: enum
        enum: [linux, macos]
        default: linux
    executor:
      name: base
      size: sm
    steps:
      - greet:
          loud: tr
      - run:
          command: ec

workflows:
  main:
    jobs:
      - build:
          os: li
          context: gl
`
	// at is what's offered at the end of the line whose text, trimmed, is
	// the given text.
	at := func(text string) []string {
		lines := strings.Split(config, "\n")
		line := slices.IndexFunc(lines, func(l string) bool { return strings.TrimSpace(l) == text })
		assert.Assert(t, line != -1, "no line %q", text)
		return completionLabels(t, config, protocol.Position{Line: uint32(line), Character: uint32(len(lines[line]))})
	}

	t.Run("an executor's enum parameter is offered its values", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(at("size: sm"), []string{"small", "medium", "large"}))
	})

	t.Run("a command's boolean parameter is offered true and false", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(at("loud: tr"), []string{"true", "false"}))
	})

	t.Run("a job's enum parameter is offered its values", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(at("os: li"), []string{"linux", "macos"}))
	})

	t.Run("a key that isn't a parameter is offered nothing", func(t *testing.T) {
		assert.Check(t, cmp.Len(at("command: ec"), 0))
		assert.Check(t, cmp.Len(at("context: gl"), 0))
	})
}
