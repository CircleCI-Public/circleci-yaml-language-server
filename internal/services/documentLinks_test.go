package languageservice

import (
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestDocumentLinksToTheOrbRegistry(t *testing.T) {
	const config = `version: 2.1

orbs:
  node: circleci/node@5.0.0
  go: circleci/go@1
`
	docURI := uri.File("/repo/.circleci/config.yml")
	c := cache.New()
	c.FileCache.SetFile(cache.File{TextDocument: protocol.TextDocumentItem{URI: docURI, Text: config}})

	targets := func(t *testing.T, hostURL string) []string {
		t.Helper()
		settings := testHelpers.DefaultSettings()
		settings.Api.HostUrl = hostURL
		links, err := DocumentLinks(protocol.DocumentLinkParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		}, c, settings)
		assert.NilError(t, err)

		found := []string{}
		for _, link := range links {
			found = append(found, string(*link.Target))
		}
		return found
	}

	t.Run("on circleci.com, at the orb's version when it is a whole one", func(t *testing.T) {
		got := targets(t, circleci.DefaultHostURL)
		assert.Check(t, cmp.DeepEqual(got, []string{
			"https://circleci.com/developer/orbs/orb/circleci/node?version=5.0.0",
			"https://circleci.com/developer/orbs/orb/circleci/go",
		}))
	})

	t.Run("not on a server of its own", func(t *testing.T) {
		got := targets(t, "https://circleci.acme.example")
		assert.Check(t, cmp.Len(got, 0))
	})
}
