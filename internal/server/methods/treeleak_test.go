package methods

import (
	"context"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/tsalloc"
)

const leakConfig = `version: 2.1

jobs:
  build:
    machine:
      image: ubuntu-2404:current
    steps:
      - checkout

workflows:
  main:
    jobs:
      - build
`

// The method layer parses documents of its own — the open document, an orb
// file being edited, and a document handed over by a command — and must close
// each one, since the tree-sitter bindings free a tree only when it is closed.
func TestMethodsCloseTheTreesTheyParse(t *testing.T) {
	fake := fakes.NewCircleCI(t)
	methods := &Methods{
		Ctx:   context.Background(),
		Cache: cache.New(),
		Settings: &session.Settings{
			Api: circleci.Config{HostUrl: fake.URL()},
		},
	}

	configURI := uri.File("/workspace/.circleci/config.yml")
	config := protocol.TextDocumentItem{URI: configURI, Text: leakConfig}
	methods.Cache.FileCache.SetFile(cache.File{TextDocument: config})

	// An orb file is recognised by the cache holding its orb id, which is
	// read from the file's last two path segments.
	orbURI := uri.File("/orbs/circleci/go.yml")
	methods.Cache.OrbCache.SetOrb(&ast.OrbInfo{}, "circleci/go")

	t.Run("parsing the open document", func(t *testing.T) {
		leaked := tsalloc.Track(t)

		methods.parsingMethods(config)

		assert.Check(t, cmp.Equal(leaked(), int64(0)), "tree-sitter allocations left open")
	})

	t.Run("updating an orb file", func(t *testing.T) {
		leaked := tsalloc.Track(t)

		methods.updateOrbFile([]byte("version: 2.1\ncommands:\n  greet:\n    steps:\n      - run: echo hello\n"), orbURI)

		assert.Check(t, cmp.Equal(leaked(), int64(0)), "tree-sitter allocations left open")
	})

	t.Run("listing workflows for a command", func(t *testing.T) {
		leaked := tsalloc.Track(t)

		params, err := protocol.Marshal(map[string]any{
			"command":   "getWorkflows",
			"arguments": []any{leakConfig, configURI.FsPath()},
		})
		assert.NilError(t, err)

		_, err = methods.ExecuteCommand(params)
		assert.NilError(t, err)

		assert.Check(t, cmp.Equal(leaked(), int64(0)), "tree-sitter allocations left open")
	})
}
