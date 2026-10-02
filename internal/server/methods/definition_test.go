package methods

import (
	"path/filepath"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

const definitionConfig = `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo build

workflows:
  main:
    jobs:
      - build
`

func TestDefinition(t *testing.T) {
	// The job's name in the workflow, and the job it names.
	jobName := protocol.Range{
		Start: protocol.Position{Line: 12, Character: 8},
		End:   protocol.Position{Line: 12, Character: 13},
	}
	job := protocol.Range{
		Start: protocol.Position{Line: 3, Character: 2},
		End:   protocol.Position{Line: 7, Character: 23},
	}
	jobKey := protocol.Range{
		Start: protocol.Position{Line: 3, Character: 2},
		End:   protocol.Position{Line: 3, Character: 7},
	}

	definitionWith := func(t *testing.T, capabilities protocol.ClientCapabilities, pos protocol.Position) (protocol.DefinitionResult, uri.URI) {
		t.Helper()
		methods := New(t.Context(), nil, cache.New(), session.Settings{}, "")
		_, err := methods.Initialize(t.Context(), &protocol.InitializeParams{Capabilities: capabilities})
		assert.NilError(t, err)

		docURI := uri.File(filepath.Join(t.TempDir(), ".circleci", "config.yml"))
		methods.Cache.FileCache.SetFile(cache.File{
			TextDocument: protocol.TextDocumentItem{URI: docURI, Text: definitionConfig},
		})

		result, err := methods.Definition(t.Context(), &protocol.DefinitionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
				Position:     pos,
			},
		})
		assert.NilError(t, err)
		return result, docURI
	}

	yes, no := true, false
	linkSupport := func(support *bool) protocol.ClientCapabilities {
		return protocol.ClientCapabilities{TextDocument: &protocol.TextDocumentClientCapabilities{
			Definition: &protocol.DefinitionClientCapabilities{LinkSupport: support},
		}}
	}

	t.Run("links from the job's name to the job for a client that supports links", func(t *testing.T) {
		result, docURI := definitionWith(t, linkSupport(&yes), protocol.Position{Line: 12, Character: 10})

		assert.Check(t, cmp.DeepEqual(result, protocol.DefinitionLinkSlice{{
			OriginSelectionRange: &jobName,
			TargetURI:            docURI,
			TargetRange:          job,
			TargetSelectionRange: jobKey,
		}}))
	})

	t.Run("gives locations to a client that doesn't support links", func(t *testing.T) {
		clients := map[string]protocol.ClientCapabilities{
			"one that says so":              linkSupport(&no),
			"one that doesn't say":          linkSupport(nil),
			"one with no definition client": {},
		}
		for name, capabilities := range clients {
			result, docURI := definitionWith(t, capabilities, protocol.Position{Line: 12, Character: 10})

			assert.Check(t, cmp.DeepEqual(result, protocol.LocationSlice{{URI: docURI, Range: job}}), name)
		}
	})

	t.Run("answers null where nothing is defined", func(t *testing.T) {
		result, _ := definitionWith(t, linkSupport(&yes), protocol.Position{Line: 0, Character: 2})

		assert.Check(t, cmp.Nil(result))
	})
}
