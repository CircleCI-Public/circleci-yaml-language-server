package circleci_test

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
)

// registryFor returns a registry pointed at a fresh fake carrying the
// circleci/go fixture.
func registryFor(t *testing.T, configure func(*fakes.CircleCI)) (circleci.OrbRegistry, *fakes.CircleCI) {
	t.Helper()

	fake := fakes.NewCircleCI(t)
	fake.SeedGoOrb()
	if configure != nil {
		configure(fake)
	}

	client := circleci.NewV3Client(circleci.Credentials{HostURL: fake.URL(), Token: "token", UserID: "user-1"}, false)

	return circleci.NewOrbRegistry(client), fake
}

func orbVersionNames(versions []circleci.OrbPackageVersion) []string {
	names := make([]string, 0, len(versions))
	for _, version := range versions {
		names = append(names, version.Version)
	}

	return names
}

func TestOrbRegistry(t *testing.T) {
	t.Run("fetches an orb and its versions", func(t *testing.T) {
		registry, _ := registryFor(t, nil)

		orb, err := registry.FetchOrb(t.Context(), "circleci/go")
		assert.NilError(t, err)
		assert.Assert(t, orb != nil)

		assert.Check(t, cmp.Equal(orb.Name, "circleci/go"))
		assert.Check(t, cmp.Equal(orb.ID, "orb-go"))

		// Upgrade hints are computed from the version list, so it has to
		// arrive with the orb.
		versions := orbVersionNames(orb.Versions)
		assert.Check(t, cmp.DeepEqual(versions, []string{
			"4.0.0", "1.12.0", "1.7.3", "1.7.1", "1.7.0", "0.1.0",
		}))
	})

	t.Run("reports an unknown orb as not found", func(t *testing.T) {
		registry, _ := registryFor(t, nil)

		orb, err := registry.FetchOrb(t.Context(), "circleci/nope")
		assert.Check(t, cmp.ErrorIs(err, circleci.ErrNotFound))
		assert.Check(t, cmp.Nil(orb))
	})

	t.Run("resolves every reference form", func(t *testing.T) {
		registry, _ := registryFor(t, nil)

		for _, testCase := range []struct {
			name string
			ref  string
			want string
		}{
			{
				name: "an exact version",
				ref:  "circleci/go@1.7.1",
				want: "1.7.1",
			},
			{
				name: "a partial minor version",
				ref:  "circleci/go@1.7",
				want: "1.7.3",
			},
			{
				name: "a partial major version",
				ref:  "circleci/go@1",
				want: "1.12.0",
			},
			{
				name: "volatile",
				ref:  "circleci/go@volatile",
				want: "4.0.0",
			},
			{
				name: "a development tag",
				ref:  "circleci/go@dev:alpha",
				want: "dev:alpha",
			},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				resolved, err := registry.ResolveVersion(t.Context(), testCase.ref)
				assert.NilError(t, err)
				assert.Assert(t, resolved != nil)

				assert.Check(t, cmp.Equal(resolved.Version, testCase.want))
				assert.Check(t, cmp.Equal(resolved.Source, "# source of "+testCase.want+"\n"))
				assert.Check(t, cmp.Equal(resolved.OrbPackageID, "orb-go"))
			})
		}
	})

	t.Run("carries sibling versions alongside a resolved version", func(t *testing.T) {
		registry, _ := registryFor(t, nil)

		resolved, err := registry.ResolveVersion(t.Context(), "circleci/go@1.7.1")
		assert.NilError(t, err)
		assert.Assert(t, resolved != nil)

		versions := orbVersionNames(resolved.Versions)
		assert.Check(t, cmp.DeepEqual(versions, []string{
			"4.0.0", "1.12.0", "1.7.3", "1.7.1", "1.7.0", "0.1.0",
		}))
	})

	t.Run("reports an unresolvable reference as not found", func(t *testing.T) {
		registry, _ := registryFor(t, nil)

		resolved, err := registry.ResolveVersion(t.Context(), "circleci/go@9.9.9")
		assert.Check(t, cmp.ErrorIs(err, circleci.ErrNotFound))
		assert.Check(t, cmp.Nil(resolved))
	})

	t.Run("finds a namespace", func(t *testing.T) {
		registry, _ := registryFor(t, nil)

		namespace, err := registry.FetchNamespace(t.Context(), "circleci")
		assert.NilError(t, err)
		assert.Assert(t, namespace != nil)

		assert.Check(t, cmp.Equal(namespace.ID, "ns-circleci"))
		assert.Check(t, cmp.Equal(namespace.Name, "circleci"))
	})

	t.Run("reports an unknown namespace as not found", func(t *testing.T) {
		registry, _ := registryFor(t, nil)

		namespace, err := registry.FetchNamespace(t.Context(), "nope")
		assert.Check(t, cmp.ErrorIs(err, circleci.ErrNotFound))
		assert.Check(t, cmp.Nil(namespace))
	})

	t.Run("lists the orbs of a namespace", func(t *testing.T) {
		registry, _ := registryFor(t, func(fake *fakes.CircleCI) {
			fake.AddOrbPackage("orb-node", "ns-circleci", "circleci", "node", false, true)
			fake.AddOrbVersion("ver-node", "orb-node", "circleci/node", "7.2.1", "", "")
			// A different namespace must not leak in.
			fake.AddNamespace("ns-other", "other")
			fake.AddOrbPackage("orb-other", "ns-other", "other", "thing", false, true)
		})

		orbs, err := registry.ListNamespaceOrbs(t.Context(), "circleci")
		assert.NilError(t, err)

		names := make([]string, 0, len(orbs))
		for _, orb := range orbs {
			names = append(names, orb.Name)
		}
		assert.Check(t, cmp.DeepEqual(names, []string{"circleci/go", "circleci/node"}))
	})

	t.Run("reports an unknown namespace when listing orbs", func(t *testing.T) {
		registry, _ := registryFor(t, nil)

		_, err := registry.ListNamespaceOrbs(t.Context(), "nope")
		assert.Check(t, cmp.ErrorIs(err, circleci.ErrNotFound))
	})
}
