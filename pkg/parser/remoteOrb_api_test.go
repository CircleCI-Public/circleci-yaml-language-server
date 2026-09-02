package parser

import (
	"net/http"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/testHelpers"
)

func TestGetOrbByName(t *testing.T) {
	t.Run("finds an orb by its qualified name", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()

		orb, err := GetOrbByName("circleci/go", testHelpers.GetLsContextForHost(fake.URL()))
		assert.NilError(t, err)

		assert.Check(t, cmp.Equal(orb.Name, "circleci/go"))
		assert.Check(t, cmp.Equal(orb.ID, "orb-go"))
	})

	t.Run("resolves public orbs without a token", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()

		lsContext := testHelpers.GetLsContextForHost(fake.URL())
		lsContext.Api.Token = ""

		orb, err := GetOrbByName("circleci/go", lsContext)
		assert.NilError(t, err)
		assert.Check(t, cmp.Equal(orb.Name, "circleci/go"))
	})

	t.Run("reports an unknown orb", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()

		_, err := GetOrbByName("circleci/nope", testHelpers.GetLsContextForHost(fake.URL()))
		assert.Check(t, cmp.ErrorContains(err, "does not exist"))
	})

	t.Run("reports a failed lookup as a failure, not as absence", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SetStatus("GET /api/v3/orb/packages", http.StatusInternalServerError)

		_, err := GetOrbByName("circleci/go", testHelpers.GetLsContextForHost(fake.URL()))
		assert.Assert(t, err != nil)
		assert.Check(t, cmp.ErrorContains(err, "500"))
	})

	t.Run("reports an unconfigured host", func(t *testing.T) {
		lsContext := testHelpers.GetLsContextForHost("")

		_, err := GetOrbByName("circleci/go", lsContext)
		assert.Assert(t, err != nil)
		assert.Check(t, cmp.ErrorContains(err, "host URL not defined"))
	})
}

func TestGetRemoteOrb(t *testing.T) {
	t.Run("resolves a reference and returns its source", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()

		orb, err := GetRemoteOrb("circleci/go@1.7.1", "token", fake.URL(), "user-1")
		assert.NilError(t, err)

		t.Run("names the resolved version", func(t *testing.T) {
			assert.Check(t, cmp.Equal(orb.Version, "1.7.1"))
			assert.Check(t, cmp.Equal(orb.Id, "ver-1-7-1"))
			assert.Check(t, cmp.Equal(orb.Orb.Id, "orb-go"))
		})

		t.Run("carries the version's YAML", func(t *testing.T) {
			assert.Check(t, cmp.Equal(orb.Source, "# source of 1.7.1\n"))
		})

		// The sibling versions are what upgrade hints are computed from.
		t.Run("carries the orb's other versions", func(t *testing.T) {
			versions := versionsOf(orb.Orb.Versions)
			assert.Check(t, cmp.DeepEqual(versions, []string{
				"4.0.0", "1.12.0", "1.7.3", "1.7.1", "1.7.0", "0.1.0",
			}))
		})
	})

	// The reach of the old GraphQL orbVersionRef argument has to be preserved,
	// or configs that used to resolve stop resolving.
	t.Run("resolves every reference form", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()

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
				orb, err := GetRemoteOrb(testCase.ref, "token", fake.URL(), "")
				assert.NilError(t, err)

				assert.Check(t, cmp.Equal(orb.Version, testCase.want))
				assert.Check(t, cmp.Equal(orb.Source, "# source of "+testCase.want+"\n"))
			})
		}
	})

	// validateSingleOrb matches on this prefix to tell "unknown version" from
	// "unknown orb", so the wording is load-bearing.
	t.Run("reports an unresolvable version with the prefix validation expects", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()

		_, err := GetRemoteOrb("circleci/go@9.9.9", "token", fake.URL(), "")
		assert.Assert(t, err != nil)

		errMessage := err.Error()
		assert.Check(t, cmp.Contains(errMessage, "could not find orb circleci/go@9.9.9"))
	})

	// Losing the version list costs upgrade hints. Losing the orb costs
	// completion, hover and go-to-definition, so the two must not be coupled.
	t.Run("still resolves when the version list cannot be fetched", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()
		fake.SetStatus("GET /api/v3/orb/packages", http.StatusInternalServerError)

		orb, err := GetRemoteOrb("circleci/go@1.7.1", "token", fake.URL(), "")
		assert.NilError(t, err)

		assert.Check(t, cmp.Equal(orb.Version, "1.7.1"))
		assert.Check(t, cmp.Equal(orb.Source, "# source of 1.7.1\n"))
		assert.Check(t, cmp.Len(orb.Orb.Versions, 0))
	})

	t.Run("fails when the source cannot be fetched", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()
		fake.SetSourceStatus("ver-1-7-1", http.StatusInternalServerError)

		_, err := GetRemoteOrb("circleci/go@1.7.1", "token", fake.URL(), "")
		assert.Assert(t, err != nil)
		assert.Check(t, cmp.ErrorContains(err, "500"))
	})

	t.Run("reports an unknown orb", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()

		_, err := GetRemoteOrb("circleci/nope@1.0.0", "token", fake.URL(), "")
		assert.Check(t, cmp.ErrorContains(err, "could not find orb"))
	})

	t.Run("reports an unconfigured host", func(t *testing.T) {
		_, err := GetRemoteOrb("circleci/go@1.7.1", "token", "", "")
		assert.Check(t, cmp.ErrorContains(err, "host URL not defined"))
	})
}

func TestGetOrbVersions(t *testing.T) {
	// The GraphQL query this replaced never selected the fields it read back,
	// so it always failed and upgrade hints never appeared for orbs served
	// from the on-disk cache. A non-empty list here is the regression test.
	t.Run("returns every published version, newest first", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()

		versions, err := GetOrbVersions("circleci/go@1.7.1", "token", fake.URL(), "")
		assert.NilError(t, err)

		got := versionsOf(versions)
		assert.Check(t, cmp.DeepEqual(got, []string{
			"4.0.0", "1.12.0", "1.7.3", "1.7.1", "1.7.0", "0.1.0",
		}))
	})

	t.Run("feeds the upgrade hints computed by GetVersionInfo", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()

		versions, err := GetOrbVersions("circleci/go@1.7.1", "token", fake.URL(), "")
		assert.NilError(t, err)

		latest, latestMinor, latestPatch := GetVersionInfo(versions, "v1.7.1")

		assert.Check(t, cmp.Equal(latest, "v4.0.0"))
		assert.Check(t, cmp.Equal(latestMinor, "v1.12.0"))
		assert.Check(t, cmp.Equal(latestPatch, "v1.7.3"))
	})

	t.Run("accepts a reference with no version", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()

		versions, err := GetOrbVersions("circleci/go", "token", fake.URL(), "")
		assert.NilError(t, err)
		assert.Check(t, cmp.Len(versions, 6))
	})

	t.Run("reports an unknown orb", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()

		_, err := GetOrbVersions("circleci/nope@1.0.0", "token", fake.URL(), "")
		assert.Check(t, cmp.ErrorContains(err, "could not find orb"))
	})

	t.Run("reports an unconfigured host", func(t *testing.T) {
		_, err := GetOrbVersions("circleci/go@1.7.1", "token", "", "")
		assert.Check(t, cmp.ErrorContains(err, "host URL not defined"))
	})
}

func versionsOf(versions []struct{ Version string }) []string {
	out := make([]string, 0, len(versions))
	for _, version := range versions {
		out = append(out, version.Version)
	}

	return out
}
