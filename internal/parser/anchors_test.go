package parser

import (
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestParseYamlAnchors(t *testing.T) {
	parse := func(t *testing.T, content string) YamlDocument {
		t.Helper()
		doc, err := ParseFromContent([]byte(content), testHelpers.DefaultSettings(), uri.File(""), protocol.Position{})
		assert.NilError(t, err)
		t.Cleanup(doc.Close)
		return doc
	}

	t.Run("an anchor is found with the aliases that refer to it", func(t *testing.T) {
		doc := parse(t, `version: 2.1
defaults: &defaults
  resource_class: large
jobs:
  build:
    <<: *defaults
  test:
    <<: *defaults
`)
		anchor, ok := doc.YamlAnchors["defaults"]
		assert.Assert(t, ok, "anchor defaults was not found")
		assert.Check(t, cmp.Len(*anchor.References, 2))
	})

	t.Run("an alias with no anchor refers to nothing", func(t *testing.T) {
		doc := parse(t, `version: 2.1
jobs:
  build:
    <<: *defaults
`)
		assert.Check(t, cmp.Len(doc.YamlAnchors, 0))
	})

	t.Run("an && in a command is not an anchor", func(t *testing.T) {
		doc := parse(t, `version: 2.1
jobs:
  build:
    steps:
      - run: make deps && make test
`)
		assert.Check(t, cmp.Len(doc.YamlAnchors, 0))
	})

	t.Run("a document without an & has no anchors", func(t *testing.T) {
		doc := parse(t, `version: 2.1
jobs:
  build:
    steps:
      - run: make test
`)
		assert.Check(t, cmp.Len(doc.YamlAnchors, 0))
	})
}
