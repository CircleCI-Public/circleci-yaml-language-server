package parser

import (
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestParseDescription(t *testing.T) {
	const config = `version: 2.1

commands:
  plain:
    description: Say hello.
    steps: [checkout]
  quoted:
    description: "Say \"hello\"."
    steps: [checkout]
  literal:
    description: |
      First line.
      Second line.
    steps: [checkout]
  folded:
    description: >-
      One long
      sentence.
    steps: [checkout]

jobs:
  build:
    description: 'Build it.'
    parameters:
      who:
        type: string
        description: "Who to greet."
    docker:
      - image: cimg/base:stable
    steps:
      - checkout
`
	doc, err := ParseFromContent([]byte(config), testHelpers.DefaultSettings(), uri.File("/config.yml"), protocol.Position{})
	assert.NilError(t, err)
	t.Cleanup(doc.Close)

	t.Run("commands", func(t *testing.T) {
		got := map[string]string{}
		for name, command := range doc.Commands {
			got[name] = command.Description
		}
		assert.Check(t, cmp.DeepEqual(got, map[string]string{
			"plain":   "Say hello.",
			"quoted":  `Say "hello".`,
			"literal": "First line.\nSecond line.",
			"folded":  "One long sentence.",
		}))
	})

	t.Run("a job and its parameter", func(t *testing.T) {
		job := doc.Jobs["build"]
		assert.Check(t, cmp.Equal(job.Description, "Build it."))
		assert.Assert(t, job.Parameters["who"] != nil)
		assert.Check(t, cmp.Equal(job.Parameters["who"].GetDescription(), "Who to greet."))
	})
}
