package hover

import (
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestParameterReference(t *testing.T) {
	const config = `version: 2.1

parameters:
  setup_go:
    type: string
    default: setup-go@v1
    description: The function that installs Go.

executors:
  vm:
    parameters:
      size:
        type: enum
        enum: [medium, large]
        default: medium
    machine:
      image: ubuntu-2204:current
    resource_class: << parameters.size >>

commands:
  greet:
    parameters:
      who:
        type: string
    steps:
      - run: echo << parameters.who >> << parameters.missing >>

jobs:
  test:
    parameters:
      os:
        type: executor
    executor: << parameters.os >>
    steps:
      - run: circleci run << pipeline.parameters.setup_go >>
      - greet:
          who: me

workflows:
  main:
    jobs:
      - test:
          os: vm
`
	doc, err := yamlparser.ParseFromContent([]byte(config), testHelpers.DefaultSettings(), uri.File("/config.yml"), protocol.Position{})
	assert.NilError(t, err)
	t.Cleanup(doc.Close)

	lines := strings.Split(config, "\n")
	at := func(line int, text string) protocol.Position {
		return protocol.Position{Line: uint32(line), Character: uint32(strings.Index(lines[line], text) + 1)}
	}

	t.Run("a job's parameter, which is required", func(t *testing.T) {
		got, ok := ParameterReference(doc, cache.New(), at(32, "parameters.os"))
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(got, "**os** `executor`\n\nA parameter of the job `test`. Required."))
	})

	t.Run("a pipeline parameter, with its default and description", func(t *testing.T) {
		got, ok := ParameterReference(doc, cache.New(), at(34, "pipeline.parameters"))
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(got,
			"**setup_go** `string`\n\nA pipeline parameter. Default: `setup-go@v1`.\n\nThe function that installs Go."))
	})

	t.Run("an executor's parameter", func(t *testing.T) {
		got, ok := ParameterReference(doc, cache.New(), at(17, "parameters.size"))
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(got, "**size** `enum`\n\nA parameter of the executor `vm`. One of `medium`, `large`. Default: `medium`."))
	})

	t.Run("a command's parameter", func(t *testing.T) {
		got, ok := ParameterReference(doc, cache.New(), at(25, "parameters.who"))
		assert.Assert(t, ok)
		assert.Check(t, cmp.Contains(got, "A parameter of the command `greet`."))
	})

	t.Run("nothing for a parameter that isn't defined", func(t *testing.T) {
		_, ok := ParameterReference(doc, cache.New(), at(25, "parameters.missing"))
		assert.Check(t, !ok)
	})

	t.Run("nothing beside a reference on the same line", func(t *testing.T) {
		_, ok := ParameterReference(doc, cache.New(), at(25, "echo"))
		assert.Check(t, !ok)
	})
}

func TestParameterReferenceInASection(t *testing.T) {
	const config = `version: 2.1

parameters:
  quiet:
    type: boolean
    default: false

commands:
  greet:
    parameters:
      loud:
        type: boolean
        default: false
    steps:
      - run: echo <<# parameters.loud >>-v<</ parameters.loud >> <<^ pipeline.parameters.quiet >>-n<</ pipeline.parameters.quiet >>
`
	doc, err := yamlparser.ParseFromContent([]byte(config), testHelpers.DefaultSettings(), uri.File("/config.yml"), protocol.Position{})
	assert.NilError(t, err)
	t.Cleanup(doc.Close)

	line := strings.Split(config, "\n")[14]
	at := func(text string) protocol.Position {
		return protocol.Position{Line: 14, Character: uint32(strings.Index(line, text) + 1)}
	}

	for _, tag := range []string{"<<# parameters.loud", "<</ parameters.loud"} {
		t.Run(tag, func(t *testing.T) {
			got, ok := ParameterReference(doc, cache.New(), at(tag))
			assert.Assert(t, ok)
			assert.Check(t, cmp.Contains(got, "A parameter of the command `greet`."))
		})
	}

	t.Run("an inverted section of a pipeline parameter", func(t *testing.T) {
		got, ok := ParameterReference(doc, cache.New(), at("<<^ pipeline.parameters.quiet"))
		assert.Assert(t, ok)
		assert.Check(t, cmp.Contains(got, "A pipeline parameter."))
	})
}
