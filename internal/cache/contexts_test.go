package cache

import (
	"net/http"
	"sync"
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

// contextNamed resolves a context by its full name.
func contextNamed(c *Cache, name string) *Context {
	return c.ContextCache.ResolveWorkflowContext(acmeOrgID, "", name)
}

func TestLoadContexts(t *testing.T) {
	t.Run("caches every page of contexts, with their variables", func(t *testing.T) {
		fake := contextFake(t)
		fake.SetPageLimit("context", 1)
		c := New()

		err := c.LoadContexts(configFor(fake.URL()), acmeOrgID)
		assert.NilError(t, err)

		deploy := contextNamed(c, "acme/deploy")
		assert.Assert(t, deploy != nil, "the context from the first page must be cached")
		assert.Check(t, cmp.Equal(deploy.Id, "ctx-deploy"))
		assert.Check(t, cmp.DeepEqual(deploy.envVariables, []string{"DEPLOY_KEY", "SLACK_WEBHOOK"}))
		assert.Check(t, contextNamed(c, "acme/build") != nil, "the context from the second page must be cached")

		// One request per page, the variables carried on each.
		requestCount := fake.RequestCount(http.MethodGet, "/api/v2/context")
		assert.Check(t, cmp.Equal(requestCount, 2))
	})

	// Every edit of a config asks for its organization's contexts, which used
	// to list them all twice over each time.
	t.Run("lists an organization once for every caller", func(t *testing.T) {
		fake := contextFake(t)
		c := New()

		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() {
				assert.Check(t, c.LoadContexts(configFor(fake.URL()), acmeOrgID))
			})
		}
		wg.Wait()

		requestCount := fake.RequestCount(http.MethodGet, "/api/v2/context")
		assert.Check(t, cmp.Equal(requestCount, 1))
	})

	// A token that can list contexts cannot always read their variables, and
	// is then left with the contexts alone rather than with nothing.
	t.Run("lists without the variables when they are refused", func(t *testing.T) {
		fake := contextFake(t)
		fake.RefuseContextEnvVars()
		c := New()

		err := c.LoadContexts(configFor(fake.URL()), acmeOrgID)
		assert.NilError(t, err)

		deploy := contextNamed(c, "acme/deploy")
		assert.Assert(t, deploy != nil)
		assert.Check(t, cmp.Len(deploy.envVariables, 0))
		assert.Check(t, c.ContextCache.IsOrganizationContextListLoaded(acmeOrgID))
	})

	// Only the second page fails, which a plain error status could not express:
	// it would fail the first page too and prove nothing about the loop. A
	// listing that stops part way would report the contexts after it as
	// missing, so nothing of it is kept.
	t.Run("keeps nothing of a listing that fails part way", func(t *testing.T) {
		fake := contextFake(t)
		fake.SetPageLimit("context", 1)
		fake.FailAfter(contextRoute, 1, http.StatusInternalServerError)
		c := New()

		err := c.LoadContexts(configFor(fake.URL()), acmeOrgID)
		assert.Check(t, httpcl.HasStatusCode(err, 500), "got %v", err)

		assert.Check(t, !c.ContextCache.IsOrganizationContextListLoaded(acmeOrgID))
		assert.Check(t, cmp.Nil(contextNamed(c, "acme/deploy")))
	})

	t.Run("does not remember a failure", func(t *testing.T) {
		fake := contextFake(t)
		c := New()

		t.Run("fail the listing", func(t *testing.T) {
			fake.SetStatus(contextRoute, http.StatusInternalServerError)
			err := c.LoadContexts(configFor(fake.URL()), acmeOrgID)
			assert.Check(t, httpcl.HasStatusCode(err, 500), "got %v", err)
		})

		t.Run("check the next call lists them", func(t *testing.T) {
			fake.SetStatus(contextRoute, 0)
			err := c.LoadContexts(configFor(fake.URL()), acmeOrgID)
			assert.NilError(t, err)
			assert.Check(t, c.ContextCache.IsOrganizationContextListLoaded(acmeOrgID))
		})
	})
}

func TestContextEnvVariables(t *testing.T) {
	fake := contextFake(t)
	c := New()

	err := c.LoadContexts(configFor(fake.URL()), acmeOrgID)
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

	t.Run("reports nothing for an organization not listed", func(t *testing.T) {
		envVars := c.ContextEnvVariables("another-org", []string{"acme/deploy"})
		assert.Check(t, cmp.Len(envVars, 0))
	})
}

func TestContextNames(t *testing.T) {
	c := New()
	err := c.LoadContexts(configFor(contextFake(t).URL()), acmeOrgID)
	assert.NilError(t, err)

	t.Run("are the full names, not the short ones they are also found by", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(c.ContextCache.ContextNames(acmeOrgID), []string{"acme/build", "acme/deploy"}))
	})

	t.Run("are none for an organization whose contexts aren't remembered", func(t *testing.T) {
		assert.Check(t, cmp.Len(c.ContextCache.ContextNames("another-org"), 0))
	})
}
