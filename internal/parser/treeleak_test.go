package parser

import (
	"testing"

	"github.com/adrg/xdg"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/tsalloc"
)

// A remote orb is parsed as a document of its own, to read its commands, jobs
// and executors into the cache. Nothing cached holds a node, so the orb's tree
// is closed as soon as they are read.
func TestRemoteOrbsCloseTheTreesTheyParse(t *testing.T) {
	// Fetched orb sources are written under the user's cache directory, which
	// a test must not touch: point it somewhere of the test's own. xdg reads
	// the environment once, so it has to be told to read it again, both now
	// and once the environment is restored.
	t.Cleanup(xdg.Reload)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	xdg.Reload()

	fake := fakes.NewCircleCI(t)
	fake.SeedGoOrb()
	settings := testHelpers.SettingsForHost(fake.URL())

	t.Run("fetching an orb", func(t *testing.T) {
		leaked := tsalloc.Track(t)

		orb, err := GetOrbInfo("circleci/go@1.7.1", cache.New(), settings)
		assert.NilError(t, err)
		// The fake's source is a single comment: what matters is that it was
		// fetched and parsed.
		assert.Check(t, cmp.Equal(orb.Source, "# source of 1.7.1\n"))

		assert.Check(t, cmp.Equal(leaked(), int64(0)), "tree-sitter allocations left open")
	})

	t.Run("reading an orb already on disk", func(t *testing.T) {
		leaked := tsalloc.Track(t)

		c := cache.New()
		orb := ast.Orb{Url: ast.OrbURL{Name: "circleci/go", Version: "1.7.1"}}
		source := []byte("version: 2.1\ncommands:\n  install:\n    steps:\n      - run: echo installing\n")

		err := AddOrbToCacheWithContent(orb, uri.File(t.TempDir()+"/go.yml"), source, settings, c)
		assert.NilError(t, err)
		assert.Assert(t, c.OrbCache.GetOrb("circleci/go@1.7.1") != nil)

		assert.Check(t, cmp.Equal(leaked(), int64(0)), "tree-sitter allocations left open")
	})
}
