package cache

import (
	"net/http"
	"sync"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
)

const orbPackagesRoute = "GET /api/v3/orb/packages"

func registryFor(fake *fakes.CircleCI) circleci.OrbRegistry {
	return circleci.NewOrbRegistry(fake.URL(), testToken, "", false)
}

func orbNames(orbs []circleci.OrbPackage) []string {
	names := make([]string, 0, len(orbs))
	for _, orb := range orbs {
		names = append(names, orb.Name)
	}

	return names
}

func versionsOf(orb *circleci.OrbPackage) []string {
	versions := make([]string, 0, len(orb.Versions))
	for _, version := range orb.Versions {
		versions = append(versions, version.Version)
	}

	return versions
}

func TestOrbPackagesInNamespace(t *testing.T) {
	t.Run("lists the orbs a namespace publishes", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-circleci", "circleci")
		for _, name := range []string{"go", "node", "python"} {
			fake.AddOrbPackage("orb-"+name, "ns-circleci", "circleci", name, false, true)
			fake.AddOrbVersion("ver-"+name, "orb-"+name, "circleci/"+name, "1.0.0", "", "")
		}

		orbs, err := New().OrbPackages.InNamespace(registryFor(fake), "circleci")
		assert.NilError(t, err)
		assert.Check(t, cmp.DeepEqual(orbNames(orbs), []string{"circleci/go", "circleci/node", "circleci/python"}))
	})

	t.Run("follows pagination", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-circleci", "circleci")
		for _, name := range []string{"alpha", "bravo", "charlie", "delta", "echo"} {
			fake.AddOrbPackage("orb-"+name, "ns-circleci", "circleci", name, false, true)
			fake.AddOrbVersion("ver-"+name, "orb-"+name, "circleci/"+name, "1.0.0", "", "")
		}
		fake.SetPageLimit("orb/packages", 2)

		orbs, err := New().OrbPackages.InNamespace(registryFor(fake), "circleci")
		assert.NilError(t, err)
		assert.Check(t, cmp.Len(orbs, 5))
	})

	// Completion asks on every keystroke, so a namespace is listed once for
	// every caller.
	t.Run("lists a namespace once for every caller", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-circleci", "circleci")
		fake.AddOrbPackage("orb-go", "ns-circleci", "circleci", "go", false, true)
		registry := registryFor(fake)
		c := New()

		// Counted as a delta because the registry probes the host once, on
		// first use, to decide between V3 and GraphQL.
		_, err := c.OrbPackages.Orb(registry, "circleci/nope")
		assert.NilError(t, err)
		before := len(fake.Requests())

		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() {
				orbs, err := c.OrbPackages.InNamespace(registry, "circleci")
				assert.Check(t, err)
				assert.Check(t, cmp.Len(orbs, 1))
			})
		}
		wg.Wait()

		// One listing: the namespace, then its orbs.
		assert.Check(t, cmp.Equal(len(fake.Requests())-before, 2))
	})

	t.Run("reports an unknown namespace as none", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)

		orbs, err := New().OrbPackages.InNamespace(registryFor(fake), "nope")
		assert.NilError(t, err)
		assert.Check(t, cmp.Nil(orbs))
	})

	t.Run("does not remember a failure", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-circleci", "circleci")
		fake.AddOrbPackage("orb-go", "ns-circleci", "circleci", "go", false, true)
		c := New()

		t.Run("fail the listing", func(t *testing.T) {
			fake.SetStatus(orbPackagesRoute, http.StatusInternalServerError)
			_, err := c.OrbPackages.InNamespace(registryFor(fake), "circleci")
			assert.Check(t, cmp.ErrorContains(err, "500"))
		})

		t.Run("check the next call lists it", func(t *testing.T) {
			fake.SetStatus(orbPackagesRoute, 0)
			orbs, err := c.OrbPackages.InNamespace(registryFor(fake), "circleci")
			assert.NilError(t, err)
			assert.Check(t, cmp.Len(orbs, 1))
		})
	})
}

func TestOrbPackagesOrb(t *testing.T) {
	t.Run("returns an orb's versions newest first", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()

		orb, err := New().OrbPackages.Orb(registryFor(fake), "circleci/go")
		assert.NilError(t, err)
		assert.Assert(t, orb != nil)
		assert.Check(t, cmp.DeepEqual(versionsOf(orb), []string{
			"4.0.0", "1.12.0", "1.7.3", "1.7.1", "1.7.0", "0.1.0",
		}))
	})

	t.Run("resolves public orbs without a token", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()

		orb, err := New().OrbPackages.Orb(circleci.NewOrbRegistry(fake.URL(), "", "", false), "circleci/go")
		assert.NilError(t, err)
		assert.Check(t, orb != nil)
	})

	t.Run("asks once for an orb", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()
		registry := registryFor(fake)
		c := New()

		_, err := c.OrbPackages.Orb(registry, "circleci/go")
		assert.NilError(t, err)
		before := len(fake.Requests())

		_, err = c.OrbPackages.Orb(registry, "circleci/go")
		assert.NilError(t, err)
		assert.Check(t, cmp.Equal(len(fake.Requests()), before), "a remembered orb must not reach the network")
	})

	// Listing a namespace already returns each orb's versions, so completing a
	// version straight after completing a name should cost nothing.
	t.Run("reuses an orb read by a namespace listing", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()
		registry := registryFor(fake)
		c := New()

		_, err := c.OrbPackages.InNamespace(registry, "circleci")
		assert.NilError(t, err)
		before := len(fake.Requests())

		orb, err := c.OrbPackages.Orb(registry, "circleci/go")
		assert.NilError(t, err)
		assert.Assert(t, orb != nil)
		assert.Check(t, cmp.Equal(orb.Name, "circleci/go"))
		assert.Check(t, cmp.Equal(len(fake.Requests()), before), "the orb was already in hand")
	})

	t.Run("reports an unknown orb as none", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()

		orb, err := New().OrbPackages.Orb(registryFor(fake), "circleci/nope")
		assert.NilError(t, err)
		assert.Check(t, cmp.Nil(orb))
	})

	t.Run("reports a failed lookup as a failure, not as absence", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SetStatus(orbPackagesRoute, http.StatusInternalServerError)

		_, err := New().OrbPackages.Orb(registryFor(fake), "circleci/go")
		assert.Check(t, cmp.ErrorContains(err, "500"))
	})

	t.Run("reports an unconfigured host", func(t *testing.T) {
		_, err := New().OrbPackages.Orb(circleci.NewOrbRegistry("", testToken, "", false), "circleci/go")
		assert.Check(t, cmp.ErrorContains(err, "host URL not defined"))
	})
}
