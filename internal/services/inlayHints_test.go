package languageservice

import (
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestInlayHintsForOrbs(t *testing.T) {
	const config = `version: 2.1

orbs:
  node: circleci/node@5
  local:
    commands:
      hello:
        steps:
          - run: echo hello
  remote: https://example.com/orbs/tools.yml
`
	docURI := uri.File("/repo/.circleci/config.yml")
	c := cache.New()
	c.FileCache.SetFile(cache.File{TextDocument: protocol.TextDocumentItem{URI: docURI, Text: config}})

	labels := func(t *testing.T) []string {
		t.Helper()
		hints, err := InlayHints(protocol.InlayHintParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
			Range:        protocol.Range{End: protocol.Position{Line: 99}},
		}, c, testHelpers.DefaultSettings())
		assert.NilError(t, err)

		found := []string{}
		for _, hint := range hints {
			label, _ := hint.Label.(protocol.String)
			found = append(found, string(label))
		}
		return found
	}

	t.Run("none for an orb that hasn't been fetched", func(t *testing.T) {
		got := labels(t)
		assert.Check(t, cmp.Len(got, 0))
	})

	t.Run("the version once it has", func(t *testing.T) {
		c.OrbCache.SetOrb(&ast.OrbInfo{RemoteInfo: ast.RemoteOrbInfo{Version: "5.4.1"}}, "circleci/node@5")

		got := labels(t)
		assert.Check(t, cmp.DeepEqual(got, []string{"→ 5.4.1"}))
	})
}
