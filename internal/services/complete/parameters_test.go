package complete

import (
	"testing"

	"go.lsp.dev/protocol"
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

func TestCompleteParameterReferences(t *testing.T) {
	const config = `version: 2.1

parameters:
  deploy:
    type: boolean
    default: false

jobs:
  build:
    parameters:
      os:
        type: string
        description: The OS to build for
        default: linux
      go-version:
        type: string
        default: "1.25"
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo << parameters.
      - run: echo << parameters.go-
      - run: echo << parameters.os == "linux" and parameters. >>
      - run: echo <<# parameters.
      - run: echo <<# parameters.os >>-v<</ parameters.

workflows:
  main:
    when: << pipeline.git.branch == "main" and pipeline.parameters.
    jobs:
      - build
`
	t.Run("after parameters., the job's parameters are offered", func(t *testing.T) {
		items := completionItemsAfter(t, config, "echo << parameters.")
		assert.Check(t, cmp.DeepEqual(labelsOf(items), []string{"go-version", "os"}))

		os := findItem(t, items, "os")
		assert.Check(t, cmp.Equal(os.InsertText, protocol.NewOptional("os >>")))
		assert.Check(t, cmp.Equal(os.Detail, protocol.NewOptional("string")))
		assert.Check(t, cmp.DeepEqual(os.Documentation,
			&protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: "The OS to build for"}))
	})

	t.Run("a name with a hyphen is still being written", func(t *testing.T) {
		got := labelsOf(completionItemsAfter(t, config, "echo << parameters.go-"))
		assert.Check(t, cmp.DeepEqual(got, []string{"go-version", "os"}))
	})

	t.Run("in an expression, a parameter that's already closed isn't closed again", func(t *testing.T) {
		items := completionItemsAfter(t, config, `"linux" and parameters.`)
		os := findItem(t, items, "os")
		assert.Check(t, cmp.Equal(os.InsertText, protocol.NewOptional("os")))
	})

	t.Run("a section's tags offer the job's parameters", func(t *testing.T) {
		opening := labelsOf(completionItemsAfter(t, config, "echo <<# parameters."))
		assert.Check(t, cmp.DeepEqual(opening, []string{"go-version", "os"}))
		closing := labelsOf(completionItemsAfter(t, config, "-v<</ parameters."))
		assert.Check(t, cmp.DeepEqual(closing, []string{"go-version", "os"}))
	})

	t.Run("in an expression, the pipeline parameters are offered", func(t *testing.T) {
		got := labelsOf(completionItemsAfter(t, config, "and pipeline.parameters."))
		assert.Check(t, cmp.DeepEqual(got, []string{"deploy"}))
	})
}
