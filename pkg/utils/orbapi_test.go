package utils_test

import (
	"context"
	"net/http"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

// goOrbFake builds a fake carrying the shared circleci/go orb fixture.
func goOrbFake(t *testing.T) *fakes.CircleCI {
	t.Helper()

	fake := fakes.NewCircleCI(t)
	fake.SeedGoOrb()

	return fake
}

func TestOrbPackageName(t *testing.T) {
	for _, testCase := range []struct {
		ref  string
		want string
	}{
		{"circleci/go@1.7.1", "circleci/go"},
		{"circleci/go@volatile", "circleci/go"},
		{"circleci/go@dev:alpha", "circleci/go"},
		{"circleci/go", "circleci/go"},
		{"", ""},
	} {
		got := utils.OrbPackageName(testCase.ref)
		assert.Check(t, cmp.Equal(got, testCase.want), "ref %q", testCase.ref)
	}
}

func TestSortOrbVersionsDesc(t *testing.T) {
	t.Run("orders releases newest first", func(t *testing.T) {
		versions := []utils.OrbPackageVersion{
			{Version: "1.7.1"},
			{Version: "4.0.0"},
			{Version: "0.1.0"},
			{Version: "1.12.0"},
			{Version: "1.7.3"},
		}

		utils.SortOrbVersionsDesc(versions)

		ordered := versionStrings(versions)
		assert.Check(t, cmp.DeepEqual(ordered, []string{"4.0.0", "1.12.0", "1.7.3", "1.7.1", "0.1.0"}))
	})

	// Callers read the first entry as "the latest", so anything that is not a
	// comparable release has to sort behind everything that is.
	t.Run("sorts unparseable versions last", func(t *testing.T) {
		versions := []utils.OrbPackageVersion{
			{Version: "dev:alpha"},
			{Version: "1.7.1"},
			{Version: "not-a-version"},
			{Version: "4.0.0"},
		}

		utils.SortOrbVersionsDesc(versions)

		ordered := versionStrings(versions)
		assert.Check(t, cmp.DeepEqual(ordered, []string{"4.0.0", "1.7.1", "dev:alpha", "not-a-version"}))
	})
}

func TestFetchOrbPackage(t *testing.T) {
	t.Run("returns the orb and its versions", func(t *testing.T) {
		fake := goOrbFake(t)
		client := utils.NewV3Client(fake.URL(), "", "", false)

		orb, err := utils.FetchOrbPackage(context.Background(), client, "circleci/go")
		assert.NilError(t, err)
		assert.Assert(t, orb != nil)

		t.Run("identifies the package", func(t *testing.T) {
			assert.Check(t, cmp.Equal(orb.ID, "orb-go"))
			assert.Check(t, cmp.Equal(orb.Name, "circleci/go"))
			assert.Check(t, cmp.Equal(orb.NamespaceID, "ns-circleci"))
			assert.Check(t, cmp.Equal(orb.IsPrivate, false))
		})

		// One request has to answer "does it exist", "what is its id" and
		// "what versions does it have", because the callers need all three.
		t.Run("carries every release, newest first", func(t *testing.T) {
			ordered := versionStrings(orb.Versions)
			assert.Check(t, cmp.DeepEqual(ordered, []string{
				"4.0.0", "1.12.0", "1.7.3", "1.7.1", "1.7.0", "0.1.0",
			}))
		})

		t.Run("costs a single request", func(t *testing.T) {
			requestCount := fake.RequestCount(http.MethodGet, "/api/v3/orb/packages")
			assert.Check(t, cmp.Equal(requestCount, 1))
		})

		t.Run("queries by fully qualified name", func(t *testing.T) {
			requests := fake.Requests()
			assert.Assert(t, cmp.Len(requests, 1))
			assert.Check(t, cmp.Equal(requests[0].Query["filter[name]"], "circleci/go"))
		})
	})

	// This route reports absence as a 200 with an empty list, unlike
	// /namespaces, which 404s.
	t.Run("reports an unknown orb as not found", func(t *testing.T) {
		fake := goOrbFake(t)
		client := utils.NewV3Client(fake.URL(), "", "", false)

		orb, err := utils.FetchOrbPackage(context.Background(), client, "circleci/nope")
		assert.Check(t, cmp.ErrorIs(err, utils.ErrNotFound))
		assert.Check(t, cmp.Nil(orb))
	})

	t.Run("distinguishes a failed request from an absent orb", func(t *testing.T) {
		fake := goOrbFake(t)
		fake.SetStatus("GET /api/v3/orb/packages", http.StatusInternalServerError)

		client := utils.NewV3Client(fake.URL(), "", "", false)

		_, err := utils.FetchOrbPackage(context.Background(), client, "circleci/go")
		assert.Assert(t, err != nil)

		isNotFound := utils.IsNotFound(err)
		assert.Check(t, !isNotFound)
	})

	t.Run("reports a private orb", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-acme", "acme")
		fake.AddOrbPackage("orb-secret", "ns-acme", "acme", "secret", true, false)

		client := utils.NewV3Client(fake.URL(), "", "", false)

		orb, err := utils.FetchOrbPackage(context.Background(), client, "acme/secret")
		assert.NilError(t, err)
		assert.Assert(t, orb != nil)

		assert.Check(t, cmp.Equal(orb.IsPrivate, true))
		assert.Check(t, cmp.Equal(orb.IsListed, false))
	})

	t.Run("handles an orb with no published versions", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-acme", "acme")
		fake.AddOrbPackage("orb-empty", "ns-acme", "acme", "empty", false, true)

		client := utils.NewV3Client(fake.URL(), "", "", false)

		orb, err := utils.FetchOrbPackage(context.Background(), client, "acme/empty")
		assert.NilError(t, err)
		assert.Assert(t, orb != nil)
		assert.Check(t, cmp.Len(orb.Versions, 0))
	})
}

func TestResolveOrbRef(t *testing.T) {
	// Reference resolution has to keep the reach the GraphQL orbVersionRef
	// argument had, or configs that used to work stop resolving.
	t.Run("resolves every reference form", func(t *testing.T) {
		fake := goOrbFake(t)
		client := utils.NewV3Client(fake.URL(), "", "", false)

		for _, testCase := range []struct {
			name string
			ref  string
			want string
		}{
			{"an exact version", "circleci/go@1.7.1", "1.7.1"},
			{"a partial minor version", "circleci/go@1.7", "1.7.3"},
			{"a partial major version", "circleci/go@1", "1.12.0"},
			{"volatile", "circleci/go@volatile", "4.0.0"},
			{"a development tag", "circleci/go@dev:alpha", "dev:alpha"},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				resolved, err := utils.ResolveOrbRef(context.Background(), client, testCase.ref, "")
				assert.NilError(t, err)
				assert.Assert(t, resolved != nil)

				assert.Check(t, cmp.Equal(resolved.Version, testCase.want))
				assert.Check(t, cmp.Equal(resolved.OrbPackageID, "orb-go"))
			})
		}
	})

	t.Run("reports an unresolvable version as not found", func(t *testing.T) {
		fake := goOrbFake(t)
		client := utils.NewV3Client(fake.URL(), "", "", false)

		resolved, err := utils.ResolveOrbRef(context.Background(), client, "circleci/go@9.9.9", "")
		assert.Check(t, cmp.ErrorIs(err, utils.ErrNotFound))
		assert.Check(t, cmp.Nil(resolved))
	})

	t.Run("reports an unknown orb as not found", func(t *testing.T) {
		fake := goOrbFake(t)
		client := utils.NewV3Client(fake.URL(), "", "", false)

		_, err := utils.ResolveOrbRef(context.Background(), client, "circleci/nope@1.0.0", "")
		assert.Check(t, cmp.ErrorIs(err, utils.ErrNotFound))
	})

	// filter[orb_id] is documented as required even though the API answers
	// without it, so it is sent whenever the caller already knows the id.
	t.Run("narrows by orb id when one is known", func(t *testing.T) {
		fake := goOrbFake(t)
		client := utils.NewV3Client(fake.URL(), "", "", false)

		resolved, err := utils.ResolveOrbRef(context.Background(), client, "circleci/go@1.7.1", "orb-go")
		assert.NilError(t, err)
		assert.Assert(t, resolved != nil)
		assert.Check(t, cmp.Equal(resolved.Version, "1.7.1"))

		requests := fake.Requests()
		assert.Assert(t, cmp.Len(requests, 1))
		assert.Check(t, cmp.DeepEqual(requests[0].Query, map[string]string{
			"filter[ref]":    "circleci/go@1.7.1",
			"filter[orb_id]": "orb-go",
		}))
	})

	t.Run("omits the orb id filter when none is known", func(t *testing.T) {
		fake := goOrbFake(t)
		client := utils.NewV3Client(fake.URL(), "", "", false)

		_, err := utils.ResolveOrbRef(context.Background(), client, "circleci/go@1.7.1", "")
		assert.NilError(t, err)

		requests := fake.Requests()
		assert.Assert(t, cmp.Len(requests, 1))
		assert.Check(t, cmp.DeepEqual(requests[0].Query, map[string]string{
			"filter[ref]": "circleci/go@1.7.1",
		}))
	})
}

func TestFetchOrbSource(t *testing.T) {
	t.Run("returns the YAML of a version", func(t *testing.T) {
		fake := goOrbFake(t)
		client := utils.NewV3Client(fake.URL(), "", "", false)

		source, err := utils.FetchOrbSource(context.Background(), client, "ver-1-7-1")
		assert.NilError(t, err)
		assert.Check(t, cmp.Equal(source, "# source of 1.7.1\n"))
	})

	t.Run("reports an unknown version as not found", func(t *testing.T) {
		fake := goOrbFake(t)
		client := utils.NewV3Client(fake.URL(), "", "", false)

		_, err := utils.FetchOrbSource(context.Background(), client, "ver-nope")
		assert.Check(t, cmp.ErrorIs(err, utils.ErrNotFound))
	})

	t.Run("propagates a server error", func(t *testing.T) {
		fake := goOrbFake(t)
		fake.SetSourceStatus("ver-1-7-1", http.StatusInternalServerError)

		client := utils.NewV3Client(fake.URL(), "", "", false)

		_, err := utils.FetchOrbSource(context.Background(), client, "ver-1-7-1")
		assert.Assert(t, err != nil)

		isNotFound := utils.IsNotFound(err)
		assert.Check(t, !isNotFound)
	})
}

func TestFetchNamespace(t *testing.T) {
	t.Run("resolves a name to an id", func(t *testing.T) {
		fake := goOrbFake(t)
		client := utils.NewV3Client(fake.URL(), "", "", false)

		namespace, err := utils.FetchNamespace(context.Background(), client, "circleci")
		assert.NilError(t, err)
		assert.Assert(t, namespace != nil)

		assert.Check(t, cmp.Equal(namespace.ID, "ns-circleci"))
		assert.Check(t, cmp.Equal(namespace.Name, "circleci"))
	})

	t.Run("reports an unknown namespace as not found", func(t *testing.T) {
		fake := goOrbFake(t)
		client := utils.NewV3Client(fake.URL(), "", "", false)

		namespace, err := utils.FetchNamespace(context.Background(), client, "nope")
		assert.Check(t, cmp.ErrorIs(err, utils.ErrNotFound))
		assert.Check(t, cmp.Nil(namespace))
	})

	// A namespace-exists check drives a diagnostic, so a failed request must
	// not look like proof the namespace is missing.
	t.Run("distinguishes a failed request from an absent namespace", func(t *testing.T) {
		fake := goOrbFake(t)
		fake.SetStatus("GET /api/v3/namespaces", http.StatusInternalServerError)

		client := utils.NewV3Client(fake.URL(), "", "", false)

		_, err := utils.FetchNamespace(context.Background(), client, "circleci")
		assert.Assert(t, err != nil)

		isNotFound := utils.IsNotFound(err)
		assert.Check(t, !isNotFound)
	})
}

func TestListNamespaceOrbs(t *testing.T) {
	t.Run("returns every orb in the namespace", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-circleci", "circleci")
		for _, name := range []string{"go", "node", "python"} {
			fake.AddOrbPackage("orb-"+name, "ns-circleci", "circleci", name, false, true)
			fake.AddOrbVersion("ver-"+name, "orb-"+name, "circleci/"+name, "1.0.0", "", "")
		}
		// An orb in a different namespace must not leak into the results.
		fake.AddNamespace("ns-other", "other")
		fake.AddOrbPackage("orb-other", "ns-other", "other", "thing", false, true)

		client := utils.NewV3Client(fake.URL(), "", "", false)

		orbs, err := utils.ListNamespaceOrbs(context.Background(), client, "ns-circleci")
		assert.NilError(t, err)

		names := make([]string, 0, len(orbs))
		for _, orb := range orbs {
			names = append(names, orb.Name)
		}
		assert.Check(t, cmp.DeepEqual(names, []string{"circleci/go", "circleci/node", "circleci/python"}))
	})

	t.Run("follows pagination", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-circleci", "circleci")
		for _, name := range []string{"alpha", "bravo", "charlie", "delta", "echo"} {
			fake.AddOrbPackage("orb-"+name, "ns-circleci", "circleci", name, false, true)
		}
		fake.SetPageLimit("orb/packages", 2)

		client := utils.NewV3Client(fake.URL(), "", "", false)

		orbs, err := utils.ListNamespaceOrbs(context.Background(), client, "ns-circleci")
		assert.NilError(t, err)

		assert.Check(t, cmp.Len(orbs, 5))

		requestCount := fake.RequestCount(http.MethodGet, "/api/v3/orb/packages")
		assert.Check(t, cmp.Equal(requestCount, 3))
	})

	t.Run("asks for the largest page the API allows", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-circleci", "circleci")

		client := utils.NewV3Client(fake.URL(), "", "", false)

		_, err := utils.ListNamespaceOrbs(context.Background(), client, "ns-circleci")
		assert.NilError(t, err)

		requests := fake.Requests()
		assert.Assert(t, cmp.Len(requests, 1))
		assert.Check(t, cmp.DeepEqual(requests[0].Query, map[string]string{
			"filter[namespace_id]": "ns-circleci",
			"page[limit]":          "1000",
		}))
	})

	t.Run("returns nothing for an unknown namespace", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		client := utils.NewV3Client(fake.URL(), "", "", false)

		orbs, err := utils.ListNamespaceOrbs(context.Background(), client, "ns-nope")
		assert.NilError(t, err)
		assert.Check(t, cmp.Len(orbs, 0))
	})
}

func versionStrings(versions []utils.OrbPackageVersion) []string {
	strings := make([]string, 0, len(versions))
	for _, version := range versions {
		strings = append(strings, version.Version)
	}

	return strings
}
