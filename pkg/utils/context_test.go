package utils

import (
	"net/http"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
)

const (
	acmeOrgID    = "11111111-2222-3333-4444-555555555555"
	contextRoute = "GET /api/v2/context"
)

// contextFake builds a fake carrying two contexts of the acme organization, one
// of them with environment variables.
func contextFake(t *testing.T) *fakes.CircleCI {
	t.Helper()

	fake := fakes.NewCircleCI(t)
	fake.AddContext(acmeOrgID, "ctx-deploy", "acme/deploy")
	fake.AddContextEnvVar("ctx-deploy", "DEPLOY_KEY")
	fake.AddContextEnvVar("ctx-deploy", "SLACK_WEBHOOK")
	fake.AddContext(acmeOrgID, "ctx-build", "acme/build")

	return fake
}

func Test_getContext(t *testing.T) {
	t.Run("lists the contexts of an organization", func(t *testing.T) {
		fake := contextFake(t)

		res, err := getContext(lsContextFor(fake.URL()), acmeOrgID, "", false)
		assert.NilError(t, err)
		assert.Assert(t, cmp.Len(res.Items, 2))

		assert.Check(t, cmp.Equal(res.Items[0].Name, "acme/deploy"))
		assert.Check(t, cmp.Equal(res.Items[0].ID, "ctx-deploy"))

		createdAt := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
		assert.Check(t, res.Items[0].CreatedAt.Equal(createdAt), "created_at must decode as a UTC instant")

		t.Run("querying the organization and authenticating", func(t *testing.T) {
			requests := fake.Requests()
			assert.Assert(t, cmp.Len(requests, 1))
			assert.Check(t, cmp.Equal(requests[0].Query["owner-id"], acmeOrgID))
			assert.Check(t, cmp.Equal(requests[0].CircleToken, testToken))

			// The first page is asked for without a page-token rather than
			// with an empty one.
			_, asksForAPage := requests[0].Query["page-token"]
			assert.Check(t, !asksForAPage, "the first page must be asked for without a token")
		})

		// Listing is used for validation, which only needs the names: asking
		// for environment variables can be refused for private contexts, so
		// the parameter is left off.
		t.Run("without asking for environment variables", func(t *testing.T) {
			requests := fake.Requests()
			assert.Assert(t, cmp.Len(requests, 1))
			assert.Check(t, cmp.Equal(requests[0].Query["include-env-vars"], ""))
			assert.Check(t, cmp.Len(res.Items[0].EnvironmentVariables, 0))
		})
	})

	t.Run("includes environment variables when asked", func(t *testing.T) {
		fake := contextFake(t)

		res, err := getContext(lsContextFor(fake.URL()), acmeOrgID, "", true)
		assert.NilError(t, err)
		assert.Assert(t, cmp.Len(res.Items, 2))

		names := envVarNames(res.Items[0].EnvironmentVariables)
		assert.Check(t, cmp.DeepEqual(names, []string{"DEPLOY_KEY", "SLACK_WEBHOOK"}))

		requests := fake.Requests()
		assert.Assert(t, cmp.Len(requests, 1))
		assert.Check(t, cmp.Equal(requests[0].Query["include-env-vars"], "true"))
	})

	t.Run("reports a rejected token", func(t *testing.T) {
		fake := contextFake(t)
		fake.RequireToken("a-different-token")

		res, err := getContext(lsContextFor(fake.URL()), acmeOrgID, "", false)
		assert.Check(t, cmp.ErrorContains(err, "HTTP 401"))
		assert.Check(t, cmp.Nil(res))
	})

	t.Run("reports a failing host", func(t *testing.T) {
		fake := contextFake(t)
		fake.SetStatus(contextRoute, http.StatusInternalServerError)

		res, err := getContext(lsContextFor(fake.URL()), acmeOrgID, "", false)
		assert.Check(t, cmp.ErrorContains(err, "HTTP 500"))
		assert.Check(t, cmp.Nil(res))
	})

	t.Run("reports a malformed body", func(t *testing.T) {
		fake := contextFake(t)
		fake.SetBody(contextRoute, "{")

		res, err := getContext(lsContextFor(fake.URL()), acmeOrgID, "", false)
		assert.Check(t, cmp.ErrorContains(err, "unexpected end of JSON input"))
		assert.Check(t, cmp.Nil(res))
	})

	t.Run("reports an unreachable host", func(t *testing.T) {
		fake := contextFake(t)
		lsContext := lsContextFor(fake.URL())
		fake.Close()

		res, err := getContext(lsContext, acmeOrgID, "", false)
		assert.Check(t, err != nil, "a host that is not answering must be reported")
		assert.Check(t, cmp.Nil(res))
	})
}

func TestGetAllContext(t *testing.T) {
	t.Run("caches every page of contexts", func(t *testing.T) {
		fake := contextFake(t)
		fake.SetPageLimit("context", 1)
		cache := CreateCache()

		err := GetAllContext(lsContextFor(fake.URL()), acmeOrgID, cache)
		assert.NilError(t, err)

		deploy := cache.ContextCache.GetOrganizationContext(acmeOrgID, "acme/deploy")
		assert.Check(t, deploy != nil, "the context from the first page must be cached")

		build := cache.ContextCache.GetOrganizationContext(acmeOrgID, "acme/build")
		assert.Check(t, build != nil, "the context from the second page must be cached")

		// Two pages of one, and a third request to learn there are no more.
		requestCount := fake.RequestCount(http.MethodGet, "/api/v2/context")
		assert.Check(t, cmp.Equal(requestCount, 2))
	})

	// Only the second page fails, which a plain error status could not express:
	// it would fail the first page too and prove nothing about the loop.
	t.Run("stops at a failing page, keeping what it read", func(t *testing.T) {
		fake := contextFake(t)
		fake.SetPageLimit("context", 1)
		fake.FailAfter(contextRoute, 1, http.StatusInternalServerError)
		cache := CreateCache()

		err := GetAllContext(lsContextFor(fake.URL()), acmeOrgID, cache)
		assert.Check(t, cmp.ErrorContains(err, "HTTP 500"))

		deploy := cache.ContextCache.GetOrganizationContext(acmeOrgID, "acme/deploy")
		assert.Check(t, deploy != nil, "the context from the page that succeeded must be cached")

		build := cache.ContextCache.GetOrganizationContext(acmeOrgID, "acme/build")
		assert.Check(t, cmp.Nil(build))
	})

	// Listing does not ask for environment variables, so a cached context has
	// a name and no variables until the completion path fills them in.
	t.Run("caches contexts without their environment variables", func(t *testing.T) {
		fake := contextFake(t)
		cache := CreateCache()

		err := GetAllContext(lsContextFor(fake.URL()), acmeOrgID, cache)
		assert.NilError(t, err)

		deploy := cache.ContextCache.GetOrganizationContext(acmeOrgID, "acme/deploy")
		assert.Assert(t, deploy != nil)
		assert.Check(t, cmp.Len(deploy.envVariables, 0))
	})
}

func TestGetAllContextWithEnvVars(t *testing.T) {
	t.Run("merges environment variable names onto listed contexts", func(t *testing.T) {
		fake := contextFake(t)
		lsContext := lsContextFor(fake.URL())
		cache := CreateCache()

		err := GetAllContext(lsContext, acmeOrgID, cache)
		assert.NilError(t, err)

		err = GetAllContextWithEnvVars(lsContext, acmeOrgID, cache)
		assert.NilError(t, err)

		deploy := cache.ContextCache.GetOrganizationContext(acmeOrgID, "acme/deploy")
		assert.Assert(t, deploy != nil)
		assert.Check(t, cmp.DeepEqual(deploy.envVariables, []string{"DEPLOY_KEY", "SLACK_WEBHOOK"}))

		// The second pass updates the contexts the first one cached instead of
		// replacing them, so the id and name survive.
		assert.Check(t, cmp.Equal(deploy.Id, "ctx-deploy"))
		assert.Check(t, cmp.Equal(deploy.Name, "acme/deploy"))
	})

	t.Run("caches contexts the listing never saw", func(t *testing.T) {
		fake := contextFake(t)
		cache := CreateCache()

		err := GetAllContextWithEnvVars(lsContextFor(fake.URL()), acmeOrgID, cache)
		assert.NilError(t, err)

		deploy := cache.ContextCache.GetOrganizationContext(acmeOrgID, "acme/deploy")
		assert.Assert(t, deploy != nil)
		assert.Check(t, cmp.DeepEqual(deploy.envVariables, []string{"DEPLOY_KEY", "SLACK_WEBHOOK"}))
	})

	// A token that can list contexts cannot always read their variables, and
	// the caller has to be able to tell that apart from an empty context.
	t.Run("reports a refused read", func(t *testing.T) {
		fake := contextFake(t)
		fake.SetStatus(contextRoute, http.StatusForbidden)
		cache := CreateCache()

		err := GetAllContextWithEnvVars(lsContextFor(fake.URL()), acmeOrgID, cache)
		assert.Check(t, cmp.ErrorContains(err, "HTTP 403"))
	})

	// Permission is per context, so a refusal can arrive on a later page after
	// the first one was served in full.
	t.Run("reports a page refused part way through", func(t *testing.T) {
		fake := contextFake(t)
		fake.SetPageLimit("context", 1)
		fake.FailAfter(contextRoute, 1, http.StatusForbidden)
		cache := CreateCache()

		err := GetAllContextWithEnvVars(lsContextFor(fake.URL()), acmeOrgID, cache)
		assert.Check(t, cmp.ErrorContains(err, "HTTP 403"))

		deploy := cache.ContextCache.GetOrganizationContext(acmeOrgID, "acme/deploy")
		assert.Assert(t, deploy != nil, "the context from the page that succeeded must be cached")
		assert.Check(t, cmp.DeepEqual(deploy.envVariables, []string{"DEPLOY_KEY", "SLACK_WEBHOOK"}))

		build := cache.ContextCache.GetOrganizationContext(acmeOrgID, "acme/build")
		assert.Check(t, cmp.Nil(build))
	})
}

func TestGetAllContextEnvVariables(t *testing.T) {
	fake := contextFake(t)
	lsContext := lsContextFor(fake.URL())
	cache := CreateCache()

	err := GetAllContextWithEnvVars(lsContext, acmeOrgID, cache)
	assert.NilError(t, err)

	t.Run("reports each variable with the context it came from", func(t *testing.T) {
		envVars := GetAllContextEnvVariables(cache, acmeOrgID, []string{"acme/deploy"})
		assert.Check(t, cmp.DeepEqual(envVars, []ContextEnvVariable{
			{Name: "DEPLOY_KEY", AssociatedContext: "acme/deploy"},
			{Name: "SLACK_WEBHOOK", AssociatedContext: "acme/deploy"},
		}))
	})

	// A config can name a context that does not exist, which has to be skipped
	// rather than crash the completion that asked.
	t.Run("skips a context that is not cached", func(t *testing.T) {
		envVars := GetAllContextEnvVariables(cache, acmeOrgID, []string{"acme/unknown"})
		assert.Check(t, cmp.Len(envVars, 0))
	})
}
