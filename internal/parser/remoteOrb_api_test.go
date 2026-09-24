package parser

import (
	"net/http"
	"os"
	"sync"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestDoesOrbExist(t *testing.T) {
	remoteOrb := func(name string) ast.Orb {
		return ast.Orb{Url: ast.OrbURL{Name: name, Version: "1.7.1"}}
	}

	t.Run("finds an orb by its qualified name", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()
		doc := YamlDocument{Context: testHelpers.SettingsForHost(fake.URL())}

		assert.Check(t, doc.DoesOrbExist(remoteOrb("circleci/go"), cache.New()))
	})

	t.Run("resolves public orbs without a token", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()
		settings := testHelpers.SettingsForHost(fake.URL())
		settings.Api.Token = ""
		doc := YamlDocument{Context: settings}

		assert.Check(t, doc.DoesOrbExist(remoteOrb("circleci/go"), cache.New()))
	})

	t.Run("reports an unknown orb", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()
		doc := YamlDocument{Context: testHelpers.SettingsForHost(fake.URL())}

		assert.Check(t, !doc.DoesOrbExist(remoteOrb("circleci/nope"), cache.New()))
	})

	// A lookup that fails says nothing about the orb. It used to be
	// remembered as the orb being missing, for the life of the process, so a
	// moment's outage flagged every orb in every config.
	t.Run("does not report an orb missing when the lookup fails", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()
		doc := YamlDocument{Context: testHelpers.SettingsForHost(fake.URL())}
		c := cache.New()

		t.Run("fail the lookup", func(t *testing.T) {
			fake.SetStatus("GET /api/v3/orb/packages", http.StatusInternalServerError)
			assert.Check(t, doc.DoesOrbExist(remoteOrb("circleci/nope"), c))
		})

		t.Run("check the failure was not remembered", func(t *testing.T) {
			fake.SetStatus("GET /api/v3/orb/packages", 0)
			assert.Check(t, !doc.DoesOrbExist(remoteOrb("circleci/nope"), c))
		})
	})

	t.Run("asks once for an orb named many times", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()
		doc := YamlDocument{Context: testHelpers.SettingsForHost(fake.URL())}
		c := cache.New()

		assert.Check(t, doc.DoesOrbExist(remoteOrb("circleci/go"), c))
		// Counted as a delta because the registry probes the host once, on
		// first use, to decide between V3 and GraphQL.
		before := len(fake.Requests())

		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() {
				assert.Check(t, doc.DoesOrbExist(remoteOrb("circleci/go"), c))
			})
		}
		wg.Wait()

		assert.Check(t, cmp.Equal(len(fake.Requests()), before), "a remembered orb must not be asked about again")
	})
}

func TestGetOrbInfo(t *testing.T) {
	const sourceRoute = "/api/v3/orb/versions/ver-1-7-1/source"

	t.Run("concurrent callers share one resolution", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()
		settings := testHelpers.SettingsForHost(fake.URL())
		c := cache.New()
		t.Cleanup(c.Close)

		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() {
				orb, err := GetOrbInfo("circleci/go@1.7.1", c, settings)
				assert.Check(t, err)
				assert.Check(t, cmp.Equal(orb.Source, "# source of 1.7.1\n"))
			})
		}
		wg.Wait()

		assert.Check(t, cmp.Equal(fake.RequestCount(http.MethodGet, sourceRoute), 1))
	})

	t.Run("does not remember a failure", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()
		settings := testHelpers.SettingsForHost(fake.URL())
		c := cache.New()
		t.Cleanup(c.Close)

		t.Run("fail the resolution", func(t *testing.T) {
			fake.SetSourceStatus("ver-1-7-1", http.StatusInternalServerError)
			_, err := GetOrbInfo("circleci/go@1.7.1", c, settings)
			assert.Check(t, cmp.ErrorContains(err, "500"))
		})

		t.Run("check the next call resolves it", func(t *testing.T) {
			fake.SetSourceStatus("ver-1-7-1", 0)
			orb, err := GetOrbInfo("circleci/go@1.7.1", c, settings)
			assert.NilError(t, err)
			assert.Check(t, cmp.Equal(orb.Source, "# source of 1.7.1\n"))
		})
	})

	// Go-to-definition opens the file, so it has to hold the source that was
	// fetched, and belong to this run of the server alone.
	t.Run("writes the source for go-to-definition to open", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()
		settings := testHelpers.SettingsForHost(fake.URL())
		c := cache.New()
		t.Cleanup(c.Close)

		orb, err := GetOrbInfo("circleci/go@1.7.1", c, settings)
		assert.NilError(t, err)
		path := orb.RemoteInfo.FilePath

		t.Run("check the file holds the source", func(t *testing.T) {
			content, err := os.ReadFile(path)
			assert.NilError(t, err)
			assert.Check(t, cmp.Equal(string(content), "# source of 1.7.1\n"))
		})

		t.Run("check the file is known as the orb's", func(t *testing.T) {
			orbID, ok := c.OrbIDOfSource(path)
			assert.Check(t, ok, "path %s", path)
			assert.Check(t, cmp.Equal(orbID, "circleci/go@1.7.1"))
		})

		t.Run("check closing the cache removes it", func(t *testing.T) {
			c.Close()
			_, err := os.Stat(path)
			assert.Check(t, cmp.ErrorIs(err, os.ErrNotExist))
		})
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

func versionsOf(versions []struct{ Version string }) []string {
	out := make([]string, 0, len(versions))
	for _, version := range versions {
		out = append(out, version.Version)
	}

	return out
}
