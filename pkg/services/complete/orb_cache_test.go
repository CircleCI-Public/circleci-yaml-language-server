package complete

import (
	"net/http"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
)

func orbNames(orbs []OrbData) []string {
	names := make([]string, 0, len(orbs))
	for _, orb := range orbs {
		names = append(names, orb.Name)
	}

	return names
}

func TestOrbCacheGetOrbsOfRegistry(t *testing.T) {
	t.Run("lists the orbs a namespace publishes", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-circleci", "circleci")
		for _, name := range []string{"go", "node", "python"} {
			fake.AddOrbPackage("orb-"+name, "ns-circleci", "circleci", name, false, true)
			fake.AddOrbVersion("ver-"+name, "orb-"+name, "circleci/"+name, "1.0.0", "", "")
		}

		cache := NewOrbCache()

		orbs, err := cache.GetOrbsOfRegistry("circleci", fake.URL(), "token", "user-1")
		assert.NilError(t, err)

		names := orbNames(orbs)
		assert.Check(t, cmp.DeepEqual(names, []string{"circleci/go", "circleci/node", "circleci/python"}))
	})

	// Autocomplete runs on every keystroke, so a second lookup must not reach
	// the network.
	t.Run("serves a repeat lookup from cache", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-circleci", "circleci")
		fake.AddOrbPackage("orb-go", "ns-circleci", "circleci", "go", false, true)

		cache := NewOrbCache()

		_, err := cache.GetOrbsOfRegistry("circleci", fake.URL(), "token", "")
		assert.NilError(t, err)

		// Counted as a delta because the registry probes the host once, on
		// first use, to decide between V3 and GraphQL.
		before := len(fake.Requests())

		_, err = cache.GetOrbsOfRegistry("circleci", fake.URL(), "token", "")
		assert.NilError(t, err)

		after := len(fake.Requests())
		assert.Check(t, cmp.Equal(after, before), "a cached lookup must not reach the network")
	})

	t.Run("follows pagination", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-circleci", "circleci")
		for _, name := range []string{"alpha", "bravo", "charlie", "delta", "echo"} {
			fake.AddOrbPackage("orb-"+name, "ns-circleci", "circleci", name, false, true)
			fake.AddOrbVersion("ver-"+name, "orb-"+name, "circleci/"+name, "1.0.0", "", "")
		}
		fake.SetPageLimit("orb/packages", 2)

		cache := NewOrbCache()

		orbs, err := cache.GetOrbsOfRegistry("circleci", fake.URL(), "token", "")
		assert.NilError(t, err)
		assert.Check(t, cmp.Len(orbs, 5))
	})

	t.Run("reports an unknown namespace", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		cache := NewOrbCache()

		_, err := cache.GetOrbsOfRegistry("nope", fake.URL(), "token", "")
		assert.Check(t, cmp.ErrorContains(err, "no namespace named nope"))
	})

	t.Run("does not cache a failed lookup", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-circleci", "circleci")
		fake.SetStatus("GET /api/v3/orb/packages", http.StatusInternalServerError)

		cache := NewOrbCache()

		_, err := cache.GetOrbsOfRegistry("circleci", fake.URL(), "token", "")
		assert.Assert(t, err != nil)

		before := len(fake.Requests())

		_, err = cache.GetOrbsOfRegistry("circleci", fake.URL(), "token", "")
		assert.Assert(t, err != nil)

		after := len(fake.Requests())
		assert.Check(t, after > before, "a failure must be retried, not cached")
	})
}

func TestOrbCacheGetVersionsOfOrb(t *testing.T) {
	t.Run("lists an orb's versions newest first", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()

		cache := NewOrbCache()

		orb, err := cache.GetVersionsOfOrb("circleci/go", fake.URL(), "token", "")
		assert.NilError(t, err)
		assert.Assert(t, orb != nil)

		versions := make([]string, 0, len(orb.Versions))
		for _, version := range orb.Versions {
			versions = append(versions, version.Version)
		}
		assert.Check(t, cmp.DeepEqual(versions, []string{
			"4.0.0", "1.12.0", "1.7.3", "1.7.1", "1.7.0", "0.1.0",
		}))
	})

	t.Run("serves a repeat lookup from cache", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()

		cache := NewOrbCache()

		_, err := cache.GetVersionsOfOrb("circleci/go", fake.URL(), "token", "")
		assert.NilError(t, err)

		before := len(fake.Requests())

		_, err = cache.GetVersionsOfOrb("circleci/go", fake.URL(), "token", "")
		assert.NilError(t, err)

		after := len(fake.Requests())
		assert.Check(t, cmp.Equal(after, before), "a cached lookup must not reach the network")
	})

	// Listing a namespace already returns each orb's versions, so completing a
	// version straight after completing a name should cost nothing.
	t.Run("reuses versions already loaded by a namespace listing", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()

		cache := NewOrbCache()

		_, err := cache.GetOrbsOfRegistry("circleci", fake.URL(), "token", "")
		assert.NilError(t, err)

		before := len(fake.Requests())

		orb, err := cache.GetVersionsOfOrb("circleci/go", fake.URL(), "token", "")
		assert.NilError(t, err)
		assert.Assert(t, orb != nil)
		assert.Check(t, cmp.Equal(orb.Name, "circleci/go"))

		after := len(fake.Requests())
		assert.Check(t, cmp.Equal(after, before), "the version list was already in hand")
	})

	t.Run("reports an unknown orb", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()

		cache := NewOrbCache()

		_, err := cache.GetVersionsOfOrb("circleci/nope", fake.URL(), "token", "")
		assert.Check(t, cmp.ErrorContains(err, "no orb named circleci/nope"))
	})

	t.Run("reports a failed lookup as a failure, not as absence", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SetStatus("GET /api/v3/orb/packages", http.StatusInternalServerError)

		cache := NewOrbCache()

		_, err := cache.GetVersionsOfOrb("circleci/go", fake.URL(), "token", "")
		assert.Assert(t, err != nil)
		assert.Check(t, cmp.ErrorContains(err, "500"))
	})
}

func TestGetOrbNameCompletions(t *testing.T) {
	t.Run("suggests each orb at its newest version", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()
		fake.AddOrbPackage("orb-node", "ns-circleci", "circleci", "node", false, true)
		fake.AddOrbVersion("ver-node-1", "orb-node", "circleci/node", "1.0.0", "", "")
		fake.AddOrbVersion("ver-node-2", "orb-node", "circleci/node", "7.2.1", "", "")

		// getOrbNameCompletions reads the package-level cache, so it has to be
		// emptied for the test rather than replaced.
		orbCache = NewOrbCache()

		completions, err := getOrbNameCompletions("circleci/", fake.URL(), "token", "")
		assert.NilError(t, err)

		assert.Check(t, cmp.DeepEqual(completions, []string{
			"circleci/go@4.0.0",
			"circleci/node@7.2.1",
		}))
	})

	// An orb with nothing published has no reference to complete to, and used
	// to contribute an empty string to the completion list.
	t.Run("skips an orb with no published versions", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-circleci", "circleci")
		fake.AddOrbPackage("orb-empty", "ns-circleci", "circleci", "empty", false, true)
		fake.AddOrbPackage("orb-go", "ns-circleci", "circleci", "go", false, true)
		fake.AddOrbVersion("ver-go", "orb-go", "circleci/go", "1.0.0", "", "")

		orbCache = NewOrbCache()

		completions, err := getOrbNameCompletions("circleci/", fake.URL(), "token", "")
		assert.NilError(t, err)

		assert.Check(t, cmp.DeepEqual(completions, []string{"circleci/go@1.0.0"}))
	})
}
