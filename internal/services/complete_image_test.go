package languageservice

import (
	"path/filepath"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestCompleteOutsideADockerImageValue(t *testing.T) {
	const content = "version: 2.1\n" +
		"jobs:\n" +
		"  build:\n" +
		"    docker:\n" +
		"      - image: cimg/node:17.2.0\n" +
		"    steps:\n" +
		"      - checkout\n"

	docURI := uri.File(filepath.Join(t.TempDir(), "config.yml"))
	c := cache.New()
	c.FileCache.SetFile(cache.File{
		TextDocument: protocol.TextDocumentItem{URI: docURI, Text: content},
	})

	// On the key, its colon, and the space before the value, the cursor is in
	// the image entry but not its value. Completion there used to slice out of
	// bounds, and the panic took the whole server down.
	for character := uint32(8); character <= 15; character++ {
		params := protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
				Position:     protocol.Position{Line: 4, Character: character},
			},
		}

		list, err := Complete(params, c, testHelpers.DefaultSettings())
		assert.Check(t, err, "character %d", character)
		assert.Check(t, cmp.Len(list.Items, 0), "character %d", character)
	}
}
