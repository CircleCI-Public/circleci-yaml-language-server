package cache

import (
	"net/http"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
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

func TestLoadContexts(t *testing.T) {
	t.Run("caches every page of contexts", func(t *testing.T) {
		fake := contextFake(t)
		fake.SetPageLimit("context", 1)
		c := New()

		err := c.LoadContexts(configFor(fake.URL()), acmeOrgID)
		assert.NilError(t, err)

		deploy := c.ContextCache.GetOrganizationContext(acmeOrgID, "acme/deploy")
		assert.Check(t, deploy != nil, "the context from the first page must be cached")

		build := c.ContextCache.GetOrganizationContext(acmeOrgID, "acme/build")
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
		c := New()

		err := c.LoadContexts(configFor(fake.URL()), acmeOrgID)
		assert.Check(t, httpcl.HasStatusCode(err, 500), "got %v", err)

		deploy := c.ContextCache.GetOrganizationContext(acmeOrgID, "acme/deploy")
		assert.Check(t, deploy != nil, "the context from the page that succeeded must be cached")

		build := c.ContextCache.GetOrganizationContext(acmeOrgID, "acme/build")
		assert.Check(t, cmp.Nil(build))
	})

	// Listing does not ask for environment variables, so a cached context has
	// a name and no variables until the completion path fills them in.
	t.Run("caches contexts without their environment variables", func(t *testing.T) {
		fake := contextFake(t)
		c := New()

		err := c.LoadContexts(configFor(fake.URL()), acmeOrgID)
		assert.NilError(t, err)

		deploy := c.ContextCache.GetOrganizationContext(acmeOrgID, "acme/deploy")
		assert.Assert(t, deploy != nil)
		assert.Check(t, cmp.Len(deploy.envVariables, 0))
	})
}

func TestLoadContextEnvVariables(t *testing.T) {
	t.Run("merges environment variable names onto listed contexts", func(t *testing.T) {
		fake := contextFake(t)
		api := configFor(fake.URL())
		c := New()

		err := c.LoadContexts(api, acmeOrgID)
		assert.NilError(t, err)

		err = c.LoadContextEnvVariables(api, acmeOrgID)
		assert.NilError(t, err)

		deploy := c.ContextCache.GetOrganizationContext(acmeOrgID, "acme/deploy")
		assert.Assert(t, deploy != nil)
		assert.Check(t, cmp.DeepEqual(deploy.envVariables, []string{"DEPLOY_KEY", "SLACK_WEBHOOK"}))

		// The second pass updates the contexts the first one cached instead of
		// replacing them, so the id and name survive.
		assert.Check(t, cmp.Equal(deploy.Id, "ctx-deploy"))
		assert.Check(t, cmp.Equal(deploy.Name, "acme/deploy"))
	})

	t.Run("caches contexts the listing never saw", func(t *testing.T) {
		fake := contextFake(t)
		c := New()

		err := c.LoadContextEnvVariables(configFor(fake.URL()), acmeOrgID)
		assert.NilError(t, err)

		deploy := c.ContextCache.GetOrganizationContext(acmeOrgID, "acme/deploy")
		assert.Assert(t, deploy != nil)
		assert.Check(t, cmp.DeepEqual(deploy.envVariables, []string{"DEPLOY_KEY", "SLACK_WEBHOOK"}))
	})

	// A token that can list contexts cannot always read their variables, and
	// the caller has to be able to tell that apart from an empty context.
	t.Run("reports a refused read", func(t *testing.T) {
		fake := contextFake(t)
		fake.SetStatus(contextRoute, http.StatusForbidden)
		c := New()

		err := c.LoadContextEnvVariables(configFor(fake.URL()), acmeOrgID)
		assert.Check(t, httpcl.HasStatusCode(err, 403), "got %v", err)
	})

	// Permission is per context, so a refusal can arrive on a later page after
	// the first one was served in full.
	t.Run("reports a page refused part way through", func(t *testing.T) {
		fake := contextFake(t)
		fake.SetPageLimit("context", 1)
		fake.FailAfter(contextRoute, 1, http.StatusForbidden)
		c := New()

		err := c.LoadContextEnvVariables(configFor(fake.URL()), acmeOrgID)
		assert.Check(t, httpcl.HasStatusCode(err, 403), "got %v", err)

		deploy := c.ContextCache.GetOrganizationContext(acmeOrgID, "acme/deploy")
		assert.Assert(t, deploy != nil, "the context from the page that succeeded must be cached")
		assert.Check(t, cmp.DeepEqual(deploy.envVariables, []string{"DEPLOY_KEY", "SLACK_WEBHOOK"}))

		build := c.ContextCache.GetOrganizationContext(acmeOrgID, "acme/build")
		assert.Check(t, cmp.Nil(build))
	})
}

func TestContextEnvVariables(t *testing.T) {
	fake := contextFake(t)
	api := configFor(fake.URL())
	c := New()

	err := c.LoadContextEnvVariables(api, acmeOrgID)
	assert.NilError(t, err)

	t.Run("reports each variable with the context it came from", func(t *testing.T) {
		envVars := c.ContextEnvVariables(acmeOrgID, []string{"acme/deploy"})
		assert.Check(t, cmp.DeepEqual(envVars, []ContextEnvVariable{
			{Name: "DEPLOY_KEY", AssociatedContext: "acme/deploy"},
			{Name: "SLACK_WEBHOOK", AssociatedContext: "acme/deploy"},
		}))
	})

	// A config can name a context that does not exist, which has to be skipped
	// rather than crash the completion that asked.
	t.Run("skips a context that is not cached", func(t *testing.T) {
		envVars := c.ContextEnvVariables(acmeOrgID, []string{"acme/unknown"})
		assert.Check(t, cmp.Len(envVars, 0))
	})
}
