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

// registryFor returns a registry pointed at a fresh fake carrying the
// circleci/go fixture. Capability answers are cached per host for the life of
// the process, so they are cleared for each test.
func registryFor(t *testing.T, configure func(*fakes.CircleCI)) (utils.OrbRegistry, *fakes.CircleCI) {
	t.Helper()

	utils.ResetOrbRegistryCapabilities()
	t.Cleanup(utils.ResetOrbRegistryCapabilities)

	fake := fakes.NewCircleCI(t)
	fake.SeedGoOrb()
	if configure != nil {
		configure(fake)
	}

	return utils.NewOrbRegistry(fake.URL(), "token", "user-1", false), fake
}

func orbVersionNames(versions []utils.OrbPackageVersion) []string {
	names := make([]string, 0, len(versions))
	for _, version := range versions {
		names = append(names, version.Version)
	}

	return names
}

// serverBackend disables the V3 orb routes, so the registry has to fall back to
// GraphQL exactly as it does against CircleCI Server.
func serverBackend(fake *fakes.CircleCI) {
	fake.DisableV3OrbRoutes()
}

// Every case runs against both backends, because the two have to be
// indistinguishable to callers.
func TestOrbRegistryAcrossBackends(t *testing.T) {
	for _, backend := range []struct {
		name      string
		configure func(*fakes.CircleCI)
	}{
		{"V3", nil},
		{"GraphQL fallback", serverBackend},
	} {
		t.Run(backend.name, func(t *testing.T) {
			t.Run("fetches an orb and its versions", func(t *testing.T) {
				registry, _ := registryFor(t, backend.configure)

				orb, err := registry.FetchOrb(context.Background(), "circleci/go")
				assert.NilError(t, err)
				assert.Assert(t, orb != nil)

				assert.Check(t, cmp.Equal(orb.Name, "circleci/go"))
				assert.Check(t, cmp.Equal(orb.ID, "orb-go"))

				// The GraphQL query this replaced selected only id and name, so
				// the version list never arrived and upgrade hints never fired.
				versions := orbVersionNames(orb.Versions)
				assert.Check(t, cmp.DeepEqual(versions, []string{
					"4.0.0", "1.12.0", "1.7.3", "1.7.1", "1.7.0", "0.1.0",
				}))
			})

			t.Run("reports an unknown orb as not found", func(t *testing.T) {
				registry, _ := registryFor(t, backend.configure)

				orb, err := registry.FetchOrb(context.Background(), "circleci/nope")
				assert.Check(t, cmp.ErrorIs(err, utils.ErrNotFound))
				assert.Check(t, cmp.Nil(orb))
			})

			t.Run("resolves every reference form", func(t *testing.T) {
				registry, _ := registryFor(t, backend.configure)

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
						resolved, err := registry.ResolveVersion(context.Background(), testCase.ref)
						assert.NilError(t, err)
						assert.Assert(t, resolved != nil)

						assert.Check(t, cmp.Equal(resolved.Version, testCase.want))
						assert.Check(t, cmp.Equal(resolved.Source, "# source of "+testCase.want+"\n"))
						assert.Check(t, cmp.Equal(resolved.OrbPackageID, "orb-go"))
					})
				}
			})

			t.Run("carries sibling versions alongside a resolved version", func(t *testing.T) {
				registry, _ := registryFor(t, backend.configure)

				resolved, err := registry.ResolveVersion(context.Background(), "circleci/go@1.7.1")
				assert.NilError(t, err)
				assert.Assert(t, resolved != nil)

				versions := orbVersionNames(resolved.Versions)
				assert.Check(t, cmp.DeepEqual(versions, []string{
					"4.0.0", "1.12.0", "1.7.3", "1.7.1", "1.7.0", "0.1.0",
				}))
			})

			t.Run("reports an unresolvable reference as not found", func(t *testing.T) {
				registry, _ := registryFor(t, backend.configure)

				resolved, err := registry.ResolveVersion(context.Background(), "circleci/go@9.9.9")
				assert.Check(t, cmp.ErrorIs(err, utils.ErrNotFound))
				assert.Check(t, cmp.Nil(resolved))
			})

			t.Run("finds a namespace", func(t *testing.T) {
				registry, _ := registryFor(t, backend.configure)

				namespace, err := registry.FetchNamespace(context.Background(), "circleci")
				assert.NilError(t, err)
				assert.Assert(t, namespace != nil)

				assert.Check(t, cmp.Equal(namespace.ID, "ns-circleci"))
				assert.Check(t, cmp.Equal(namespace.Name, "circleci"))
			})

			t.Run("reports an unknown namespace as not found", func(t *testing.T) {
				registry, _ := registryFor(t, backend.configure)

				namespace, err := registry.FetchNamespace(context.Background(), "nope")
				assert.Check(t, cmp.ErrorIs(err, utils.ErrNotFound))
				assert.Check(t, cmp.Nil(namespace))
			})

			t.Run("lists the orbs of a namespace", func(t *testing.T) {
				registry, _ := registryFor(t, func(fake *fakes.CircleCI) {
					if backend.configure != nil {
						backend.configure(fake)
					}
					fake.AddOrbPackage("orb-node", "ns-circleci", "circleci", "node", false, true)
					fake.AddOrbVersion("ver-node", "orb-node", "circleci/node", "7.2.1", "", "")
					// A different namespace must not leak in.
					fake.AddNamespace("ns-other", "other")
					fake.AddOrbPackage("orb-other", "ns-other", "other", "thing", false, true)
				})

				orbs, err := registry.ListNamespaceOrbs(context.Background(), "circleci")
				assert.NilError(t, err)

				names := make([]string, 0, len(orbs))
				for _, orb := range orbs {
					names = append(names, orb.Name)
				}
				assert.Check(t, cmp.DeepEqual(names, []string{"circleci/go", "circleci/node"}))
			})

			t.Run("reports an unknown namespace when listing orbs", func(t *testing.T) {
				registry, _ := registryFor(t, backend.configure)

				_, err := registry.ListNamespaceOrbs(context.Background(), "nope")
				assert.Check(t, cmp.ErrorIs(err, utils.ErrNotFound))
			})
		})
	}
}

func TestOrbRegistryBackendSelection(t *testing.T) {
	t.Run("uses V3 when the orb routes answer", func(t *testing.T) {
		registry, fake := registryFor(t, nil)

		_, err := registry.FetchOrb(context.Background(), "circleci/go")
		assert.NilError(t, err)

		graphQLRequests := fake.RequestCount(http.MethodPost, "/graphql-unstable")
		assert.Check(t, cmp.Equal(graphQLRequests, 0), "V3 answered, so GraphQL must not be used")
	})

	t.Run("falls back to GraphQL when the orb routes are absent", func(t *testing.T) {
		registry, fake := registryFor(t, serverBackend)

		orb, err := registry.FetchOrb(context.Background(), "circleci/go")
		assert.NilError(t, err)
		assert.Assert(t, orb != nil)
		assert.Check(t, cmp.Equal(orb.Name, "circleci/go"))

		graphQLRequests := fake.RequestCount(http.MethodPost, "/graphql-unstable")
		assert.Check(t, graphQLRequests > 0, "the V3 routes were absent, so GraphQL must answer")
	})

	// This is the distinction the fallback turns on. orb/packages answers a
	// missing orb with 200 and an empty list, and only answers 404 when the
	// route itself is absent. Confusing the two would send every lookup of a
	// non-existent orb down the GraphQL path.
	t.Run("does not fall back merely because an orb is missing", func(t *testing.T) {
		registry, fake := registryFor(t, nil)

		_, err := registry.FetchOrb(context.Background(), "circleci/nope")
		assert.Check(t, cmp.ErrorIs(err, utils.ErrNotFound))

		graphQLRequests := fake.RequestCount(http.MethodPost, "/graphql-unstable")
		assert.Check(t, cmp.Equal(graphQLRequests, 0), "a missing orb is not a missing route")
	})

	// Likewise a missing namespace, which this route really does report as 404.
	t.Run("does not fall back merely because a namespace is missing", func(t *testing.T) {
		registry, fake := registryFor(t, nil)

		_, err := registry.FetchNamespace(context.Background(), "nope")
		assert.Check(t, cmp.ErrorIs(err, utils.ErrNotFound))

		graphQLRequests := fake.RequestCount(http.MethodPost, "/graphql-unstable")
		assert.Check(t, cmp.Equal(graphQLRequests, 0))
	})

	t.Run("probes a host once and remembers the answer", func(t *testing.T) {
		registry, fake := registryFor(t, serverBackend)

		for range 3 {
			_, err := registry.FetchOrb(context.Background(), "circleci/go")
			assert.NilError(t, err)
		}

		probes := fake.RequestCount(http.MethodGet, "/api/v3/orb/packages")
		assert.Check(t, cmp.Equal(probes, 1), "the capability answer must be cached per host")

		graphQLRequests := fake.RequestCount(http.MethodPost, "/graphql-unstable")
		assert.Check(t, cmp.Equal(graphQLRequests, 3))
	})

	t.Run("shares the cached answer across registry instances", func(t *testing.T) {
		utils.ResetOrbRegistryCapabilities()
		t.Cleanup(utils.ResetOrbRegistryCapabilities)

		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()
		fake.DisableV3OrbRoutes()

		for range 3 {
			registry := utils.NewOrbRegistry(fake.URL(), "token", "", false)
			_, err := registry.FetchOrb(context.Background(), "circleci/go")
			assert.NilError(t, err)
		}

		probes := fake.RequestCount(http.MethodGet, "/api/v3/orb/packages")
		assert.Check(t, cmp.Equal(probes, 1))
	})

	// A failed probe says nothing about whether the route exists, so it must
	// not be remembered as an answer.
	t.Run("retries after a probe that could not settle the question", func(t *testing.T) {
		registry, fake := registryFor(t, func(fake *fakes.CircleCI) {
			fake.SetStatus("GET /api/v3/orb/packages", http.StatusInternalServerError)
		})

		for range 2 {
			_, err := registry.FetchOrb(context.Background(), "circleci/go")
			assert.Assert(t, err != nil)
		}

		probes := fake.RequestCount(http.MethodGet, "/api/v3/orb/packages")
		assert.Check(t, probes > 2, "an inconclusive probe must not be cached")
	})
}

func TestOrbRegistryGraphQLDetails(t *testing.T) {
	t.Run("sends the token unprefixed", func(t *testing.T) {
		registry, fake := registryFor(t, serverBackend)

		_, err := registry.FetchOrb(context.Background(), "circleci/go")
		assert.NilError(t, err)

		var graphQL *fakes.Request
		for _, request := range fake.Requests() {
			if request.Path == "/graphql-unstable" {
				graphQL = &request

				break
			}
		}
		assert.Assert(t, graphQL != nil)

		// The GraphQL endpoint takes a bare token, not a Bearer credential.
		assert.Check(t, cmp.Equal(graphQL.Authorization, "token"))
		assert.Check(t, cmp.Equal(graphQL.UserID, "user-1"))
	})

	t.Run("omits the authorization header when there is no token", func(t *testing.T) {
		utils.ResetOrbRegistryCapabilities()
		t.Cleanup(utils.ResetOrbRegistryCapabilities)

		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()
		fake.DisableV3OrbRoutes()

		registry := utils.NewOrbRegistry(fake.URL(), "", "", false)

		orb, err := registry.FetchOrb(context.Background(), "circleci/go")
		assert.NilError(t, err)
		assert.Assert(t, orb != nil)

		for _, request := range fake.Requests() {
			assert.Check(t, cmp.Equal(request.Authorization, ""), "path %s", request.Path)
		}
	})

	t.Run("authenticates when the host requires it", func(t *testing.T) {
		utils.ResetOrbRegistryCapabilities()
		t.Cleanup(utils.ResetOrbRegistryCapabilities)

		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()
		fake.DisableV3OrbRoutes()
		fake.RequireToken("the-real-token")

		registry := utils.NewOrbRegistry(fake.URL(), "the-real-token", "", false)

		orb, err := registry.FetchOrb(context.Background(), "circleci/go")
		assert.NilError(t, err)
		assert.Assert(t, orb != nil)
	})

	t.Run("surfaces a rejected token", func(t *testing.T) {
		utils.ResetOrbRegistryCapabilities()
		t.Cleanup(utils.ResetOrbRegistryCapabilities)

		fake := fakes.NewCircleCI(t)
		fake.SeedGoOrb()
		fake.DisableV3OrbRoutes()
		fake.RequireToken("the-real-token")

		registry := utils.NewOrbRegistry(fake.URL(), "wrong-token", "", false)

		_, err := registry.FetchOrb(context.Background(), "circleci/go")
		assert.Assert(t, err != nil)

		isNotFound := utils.IsNotFound(err)
		assert.Check(t, !isNotFound, "a rejected token is not an absent orb")
	})

	// orbs(first:) is not paged here, so a namespace larger than one page is
	// served as far as it was read rather than failing outright.
	t.Run("serves the orbs it read from a namespace it could not read in full", func(t *testing.T) {
		registry, _ := registryFor(t, func(fake *fakes.CircleCI) {
			fake.DisableV3OrbRoutes()
			fake.SetNamespaceHasMoreOrbs()
		})

		orbs, err := registry.ListNamespaceOrbs(context.Background(), "circleci")
		assert.NilError(t, err)

		names := make([]string, 0, len(orbs))
		for _, orb := range orbs {
			names = append(names, orb.Name)
		}
		assert.Check(t, cmp.DeepEqual(names, []string{"circleci/go"}))
	})

	t.Run("reports a transport failure", func(t *testing.T) {
		registry, _ := registryFor(t, func(fake *fakes.CircleCI) {
			fake.DisableV3OrbRoutes()
			fake.SetStatus("POST /graphql-unstable", http.StatusInternalServerError)
		})

		_, err := registry.FetchOrb(context.Background(), "circleci/go")
		assert.Assert(t, err != nil)

		isNotFound := utils.IsNotFound(err)
		assert.Check(t, !isNotFound, "a failed request is not an absent orb")
	})
}
