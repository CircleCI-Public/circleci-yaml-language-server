package utils_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

func TestV3ClientGet(t *testing.T) {
	t.Run("unwraps the data envelope", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-1", "circleci")

		client := utils.NewV3Client(fake.URL(), "", "", false)

		query := url.Values{"filter[name]": {"circleci"}}
		var data struct {
			ID         string `json:"id"`
			Attributes struct {
				Name string `json:"name"`
			} `json:"attributes"`
		}
		err := client.Get(context.Background(), "namespaces", query, &data)
		assert.NilError(t, err)

		assert.Check(t, cmp.Equal(data.ID, "ns-1"))
		assert.Check(t, cmp.Equal(data.Attributes.Name, "circleci"))
	})

	t.Run("sends the request the API expects", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-1", "circleci")

		client := utils.NewV3Client(fake.URL(), "secret-token", "user-42", false)

		var data struct{}
		err := client.Get(context.Background(), "namespaces", url.Values{"filter[name]": {"circleci"}}, &data)
		assert.NilError(t, err)

		requests := fake.Requests()
		assert.Assert(t, cmp.Len(requests, 1))

		t.Run("targets the versioned path", func(t *testing.T) {
			assert.Check(t, cmp.Equal(requests[0].Path, "/api/v3/namespaces"))
			assert.Check(t, cmp.Equal(requests[0].Method, http.MethodGet))
		})

		t.Run("passes the bracketed filter through unmangled", func(t *testing.T) {
			assert.Check(t, cmp.DeepEqual(requests[0].Query, map[string]string{
				"filter[name]": "circleci",
			}))
		})

		t.Run("authenticates with a bearer token", func(t *testing.T) {
			assert.Check(t, cmp.Equal(requests[0].Authorization, "Bearer secret-token"))
		})

		t.Run("identifies itself for telemetry", func(t *testing.T) {
			assert.Check(t, cmp.Equal(requests[0].UserID, "user-42"))
			assert.Check(t, cmp.Contains(requests[0].UserAgent, "CircleCI-Language-Server"))
		})
	})

	// The language server resolves public orbs for users who have not logged
	// in. Sending "Bearer " with an empty token is rejected, so the header has
	// to be left off entirely.
	t.Run("omits the authorization header when there is no token", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-1", "circleci")

		client := utils.NewV3Client(fake.URL(), "", "", false)

		var data struct{}
		err := client.Get(context.Background(), "namespaces", url.Values{"filter[name]": {"circleci"}}, &data)
		assert.NilError(t, err)

		requests := fake.Requests()
		assert.Assert(t, cmp.Len(requests, 1))
		assert.Check(t, cmp.Equal(requests[0].Authorization, ""))
	})

	t.Run("omits the telemetry header when there is no user id", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-1", "circleci")

		client := utils.NewV3Client(fake.URL(), "", "", false)

		var data struct{}
		err := client.Get(context.Background(), "namespaces", url.Values{"filter[name]": {"circleci"}}, &data)
		assert.NilError(t, err)

		requests := fake.Requests()
		assert.Assert(t, cmp.Len(requests, 1))
		assert.Check(t, cmp.Equal(requests[0].UserID, ""))
	})
}

func TestV3ClientGetErrors(t *testing.T) {
	t.Run("a 404 matches ErrNotFound", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		client := utils.NewV3Client(fake.URL(), "", "", false)

		var data struct{}
		err := client.Get(context.Background(), "namespaces", url.Values{"filter[name]": {"nope"}}, &data)
		assert.Check(t, cmp.ErrorIs(err, utils.ErrNotFound))
	})

	// Absence and unreachability have to stay distinguishable: a caller that
	// turns "missing" into a diagnostic must not fire on a broken connection.
	t.Run("a 500 does not match ErrNotFound", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SetStatus("GET /api/v3/namespaces", http.StatusInternalServerError)

		client := utils.NewV3Client(fake.URL(), "", "", false)

		var data struct{}
		err := client.Get(context.Background(), "namespaces", url.Values{"filter[name]": {"circleci"}}, &data)
		assert.Assert(t, err != nil)

		isNotFound := utils.IsNotFound(err)
		assert.Check(t, !isNotFound, "a server error must not be reported as absence")
	})

	t.Run("reports the error envelope", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		client := utils.NewV3Client(fake.URL(), "", "", false)

		// filter[name] is required, so omitting it produces a 400 carrying the
		// V3 error envelope.
		var data struct{}
		err := client.Get(context.Background(), "namespaces", nil, &data)
		assert.Assert(t, err != nil)

		assert.Check(t, cmp.ErrorContains(err, "400"))
		assert.Check(t, cmp.ErrorContains(err, "filter[name] is required"))
	})

	t.Run("a 401 is surfaced rather than swallowed", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-1", "circleci")
		fake.RequireToken("the-real-token")

		client := utils.NewV3Client(fake.URL(), "wrong-token", "", false)

		var data struct{}
		err := client.Get(context.Background(), "namespaces", url.Values{"filter[name]": {"circleci"}}, &data)
		assert.Assert(t, err != nil)
		assert.Check(t, cmp.ErrorContains(err, "401"))
	})

	t.Run("rejects an unconfigured host", func(t *testing.T) {
		client := utils.NewV3Client("", "", "", false)

		var data struct{}
		err := client.Get(context.Background(), "namespaces", nil, &data)
		assert.Check(t, cmp.ErrorIs(err, utils.ErrHostNotDefined))
	})

	t.Run("rejects a host without a scheme", func(t *testing.T) {
		client := utils.NewV3Client("circleci.com", "", "", false)

		var data struct{}
		err := client.Get(context.Background(), "namespaces", nil, &data)
		assert.Check(t, cmp.ErrorContains(err, "absolute URL"))
	})

	t.Run("honours a cancelled context", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		client := utils.NewV3Client(fake.URL(), "", "", false)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		var data struct{}
		err := client.Get(ctx, "namespaces", url.Values{"filter[name]": {"circleci"}}, &data)
		assert.Check(t, cmp.ErrorIs(err, context.Canceled))
	})
}

func TestV3ClientGetText(t *testing.T) {
	t.Run("returns a text/plain body verbatim", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddOrbPackage("orb-1", "ns-1", "circleci", "go", false, true)
		fake.AddOrbVersion("ver-1", "orb-1", "circleci/go", "1.7.1", "version: 2.1\n", "")

		client := utils.NewV3Client(fake.URL(), "", "", false)

		source, err := client.GetText(context.Background(), "orb/versions/ver-1/source", nil)
		assert.NilError(t, err)
		assert.Check(t, cmp.Equal(source, "version: 2.1\n"))
	})

	t.Run("surfaces a missing body as ErrNotFound", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		client := utils.NewV3Client(fake.URL(), "", "", false)

		source, err := client.GetText(context.Background(), "orb/versions/nope/source", nil)
		assert.Check(t, cmp.ErrorIs(err, utils.ErrNotFound))
		assert.Check(t, cmp.Equal(source, ""))
	})
}

func TestGetPaged(t *testing.T) {
	// The cursor is opaque and must be echoed back untouched, so the only way
	// to know the loop works is to make the fake hand out more than one page.
	t.Run("follows cursors until the last page", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-1", "circleci")
		for _, name := range []string{"alpha", "bravo", "charlie", "delta", "echo"} {
			fake.AddOrbPackage("orb-"+name, "ns-1", "circleci", name, false, true)
		}
		fake.SetPageLimit("orb/packages", 2)

		client := utils.NewV3Client(fake.URL(), "", "", false)

		type packageData struct {
			Attributes struct {
				Name string `json:"name"`
			} `json:"attributes"`
		}
		query := url.Values{"filter[namespace_id]": {"ns-1"}}
		packages, err := utils.GetPaged[packageData](context.Background(), client, "orb/packages", query)
		assert.NilError(t, err)

		names := make([]string, 0, len(packages))
		for _, pkg := range packages {
			names = append(names, pkg.Attributes.Name)
		}

		t.Run("collects every item across pages", func(t *testing.T) {
			assert.Check(t, cmp.DeepEqual(names, []string{
				"circleci/alpha",
				"circleci/bravo",
				"circleci/charlie",
				"circleci/delta",
				"circleci/echo",
			}))
		})

		t.Run("makes one request per page", func(t *testing.T) {
			requestCount := fake.RequestCount(http.MethodGet, "/api/v3/orb/packages")
			assert.Check(t, cmp.Equal(requestCount, 3))
		})

		t.Run("keeps the caller's filters on every page", func(t *testing.T) {
			for _, request := range fake.Requests() {
				assert.Check(t, cmp.Equal(request.Query["filter[namespace_id]"], "ns-1"),
					"page request %q dropped the filter", request.Query["page[cursor]"])
			}
		})
	})

	t.Run("does not mutate the caller's query values", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-1", "circleci")
		fake.AddOrbPackage("orb-1", "ns-1", "circleci", "alpha", false, true)
		fake.AddOrbPackage("orb-2", "ns-1", "circleci", "bravo", false, true)
		fake.SetPageLimit("orb/packages", 1)

		client := utils.NewV3Client(fake.URL(), "", "", false)

		query := url.Values{"filter[namespace_id]": {"ns-1"}}
		_, err := utils.GetPaged[struct{}](context.Background(), client, "orb/packages", query)
		assert.NilError(t, err)

		assert.Check(t, cmp.DeepEqual(query, url.Values{"filter[namespace_id]": {"ns-1"}}))
	})

	t.Run("returns an empty slice rather than nil for no matches", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		client := utils.NewV3Client(fake.URL(), "", "", false)

		query := url.Values{"filter[name]": {"circleci/nope"}}
		packages, err := utils.GetPaged[struct{}](context.Background(), client, "orb/packages", query)
		assert.NilError(t, err)

		assert.Check(t, cmp.Len(packages, 0))
		assert.Check(t, packages != nil)
	})

	t.Run("propagates an error from a later page", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-1", "circleci")
		fake.AddOrbPackage("orb-1", "ns-1", "circleci", "alpha", false, true)
		fake.AddOrbPackage("orb-2", "ns-1", "circleci", "bravo", false, true)
		fake.SetPageLimit("orb/packages", 1)
		fake.FailAfter("GET /api/v3/orb/packages", 1, http.StatusInternalServerError)

		client := utils.NewV3Client(fake.URL(), "", "", false)

		_, err := utils.GetPaged[struct{}](context.Background(), client, "orb/packages", nil)
		assert.Check(t, err != nil, "a failed later page must not be silently truncated")
	})

	t.Run("reports a malformed body", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.SetBody("GET /api/v3/orb/packages", `not json`)

		client := utils.NewV3Client(fake.URL(), "", "", false)

		_, err := utils.GetPaged[struct{}](context.Background(), client, "orb/packages", nil)
		assert.Check(t, cmp.ErrorContains(err, "decoding response"))
	})
}
