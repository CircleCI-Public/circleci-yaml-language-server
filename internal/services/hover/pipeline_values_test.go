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

func TestPipelineValue(t *testing.T) {
	const config = `version: 2.1

parameters:
  deploy:
    type: boolean
    default: false

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo << pipeline.git.branch.is_default >> << pipeline.schedule.name >>

workflows:
  main:
    when: << pipeline.parameters.deploy >>
    jobs:
      - build
`
	doc, err := yamlparser.ParseFromContent([]byte(config), testHelpers.DefaultSettings(), uri.File("/config.yml"), protocol.Position{})
	assert.NilError(t, err)
	t.Cleanup(doc.Close)

	lines := strings.Split(config, "\n")
	at := func(line int, text string) protocol.Position {
		return protocol.Position{Line: uint32(line), Character: uint32(strings.Index(lines[line], text) + 1)}
	}

	t.Run("a value shows its type and definition", func(t *testing.T) {
		got, ok := PipelineValue(doc, cache.New(), at(12, "pipeline.git"))
		assert.Assert(t, ok)
		assert.Check(t, cmp.Contains(got, "**pipeline.git.branch.is_default** `boolean`\n\n"))
	})

	t.Run("a replaced value says what replaces it", func(t *testing.T) {
		got, ok := PipelineValue(doc, cache.New(), at(12, "pipeline.schedule"))
		assert.Assert(t, ok)
		assert.Check(t, cmp.Contains(got, "Deprecated: use `pipeline.trigger.name` instead."))
	})

	t.Run("a pipeline parameter doesn't", func(t *testing.T) {
		_, ok := PipelineValue(doc, cache.New(), at(16, "pipeline.parameters"))
		assert.Check(t, !ok)
	})

	t.Run("nor does the rest of the line", func(t *testing.T) {
		_, ok := PipelineValue(doc, cache.New(), at(12, "run"))
		assert.Check(t, !ok)
	})
}
