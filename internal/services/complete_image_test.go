package languageservice

import (
	"path/filepath"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/dockerhub"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
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

func TestCompleteADockerImageFromTheConfiguredDockerHub(t *testing.T) {
	hub := fakes.NewDockerHub(t)
	hub.AddRepository("cimg", "node")
	hub.AddTag("cimg", "node", "22.1.0", "active")

	settings := testHelpers.DefaultSettings()
	settings.DockerHub = dockerhub.Config{BaseURL: hub.URL()}

	complete := func(t *testing.T, image string) []string {
		t.Helper()

		content := "version: 2.1\n" +
			"jobs:\n" +
			"  build:\n" +
			"    docker:\n" +
			"      - image: " + image + "\n" +
			"    steps:\n" +
			"      - checkout\n"

		docURI := uri.File(filepath.Join(t.TempDir(), "config.yml"))
		c := cache.New()
		c.FileCache.SetFile(cache.File{
			TextDocument: protocol.TextDocumentItem{URI: docURI, Text: content},
		})

		list, err := Complete(protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
				Position:     protocol.Position{Line: 4, Character: uint32(len("      - image: " + image))},
			},
		}, c, settings)
		assert.NilError(t, err)

		labels := make([]string, 0, len(list.Items))
		for _, item := range list.Items {
			labels = append(labels, item.Label)
		}
		return labels
	}

	t.Run("suggests its repositories", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(complete(t, "cimg/no"), []string{"cimg/node:latest"}))
	})

	t.Run("suggests its tags", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(complete(t, "cimg/node:2"), []string{"cimg/node:22.1.0"}))
	})
}

func TestDiagnosticsAskTheConfiguredDockerHub(t *testing.T) {
	hub := fakes.NewDockerHub(t)

	settings := testHelpers.DefaultSettings()
	settings.DockerHub = dockerhub.Config{BaseURL: hub.URL()}

	content := "version: 2.1\n" +
		"jobs:\n" +
		"  build:\n" +
		"    docker:\n" +
		"      - image: acme/missing:1.0\n" +
		"    steps:\n" +
		"      - checkout\n"

	_, err := DiagnosticString(content, cache.New(), settings, "")
	assert.NilError(t, err)

	paths := make([]string, 0)
	for _, request := range hub.Requests() {
		paths = append(paths, request.Path)
	}
	assert.Check(t, cmp.Contains(paths, "/v2/namespaces/acme/repositories/missing"))
}
