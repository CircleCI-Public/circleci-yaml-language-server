package parser

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/tsalloc"
)

// A remote orb is parsed as a document of its own, to read its commands, jobs
// and executors into the cache. Nothing cached holds a node, so the orb's tree
// is closed as soon as they are read.
func TestRemoteOrbsCloseTheTreesTheyParse(t *testing.T) {
	fake := fakes.NewCircleCI(t)
	fake.SeedGoOrb()
	settings := testHelpers.SettingsForHost(fake.URL())

	t.Run("fetching an orb", func(t *testing.T) {
		leaked := tsalloc.Track(t)

		c := cache.New()
		t.Cleanup(c.Close)

		orb, err := GetOrbInfo("circleci/go@1.7.1", c, settings)
		assert.NilError(t, err)
		// The fake's source is a single comment: what matters is that it was
		// fetched and parsed.
		assert.Check(t, cmp.Equal(orb.Source, "# source of 1.7.1\n"))

		assert.Check(t, cmp.Equal(leaked(), int64(0)), "tree-sitter allocations left open")
	})
}
