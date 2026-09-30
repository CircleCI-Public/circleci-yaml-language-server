package methods

import (
	"context"
	"path/filepath"
	"testing"

	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

func TestServes(t *testing.T) {
	methods := New(context.Background(), nil, cache.New(), session.Settings{}, "")

	t.Run("documents in a .circleci directory", func(t *testing.T) {
		tests := map[string]uri.URI{
			"the config":                               uri.File("/repo/.circleci/config.yml"),
			"another file beside the config":           uri.File("/repo/.circleci/deploy.yaml"),
			"a config a setup workflow continues with": uri.File("/repo/.circleci/continue/build.yml"),
			"a nested project's config":                uri.File("/repo/services/api/.circleci/config.yml"),
			"a Windows path":                           uri.URI("file:///c%3A/Users/jane/repo/.circleci/config.yml"),
		}
		for name, document := range tests {
			assert.Check(t, methods.Serves(document), "%s: %s", name, document)
		}
	})

	t.Run("other YAML", func(t *testing.T) {
		tests := map[string]uri.URI{
			"a GitHub workflow":                    uri.File("/repo/.github/workflows/ci.yml"),
			"a Compose file":                       uri.File("/repo/docker-compose.yml"),
			"a file named like the directory":      uri.File("/repo/.circleci.yml"),
			"a directory named like it in part":    uri.File("/repo/not.circleci/config.yml"),
			"a document that isn't a file on disk": uri.URI("untitled:Untitled-1"),
		}
		for name, document := range tests {
			assert.Check(t, !methods.Serves(document), "%s: %s", name, document)
		}
	})

	t.Run("an orb source the server wrote", func(t *testing.T) {
		path, err := methods.Cache.WriteOrbSource("circleci/go@1.7.1", "version: 2.1\n")
		assert.NilError(t, err)
		t.Cleanup(methods.Cache.Close)

		assert.Check(t, cmp.Equal(filepath.Ext(path), ".yml"))
		assert.Check(t, !inCircleCIDirectory(uri.File(path)), "the source is written outside any .circleci directory")
		assert.Check(t, methods.Serves(uri.File(path)), path)
	})
}
