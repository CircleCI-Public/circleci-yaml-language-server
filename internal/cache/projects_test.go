package cache

import (
	"net/http"
	"sync"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
)

const (
	rocketSlug     = "gh/acme/rocket"
	rocketEnvRoute = "GET /api/v2/project/gh/acme/rocket/envvar"
)

// projectFake builds a fake carrying the acme/rocket project and the env vars
// named, in order.
func projectFake(t *testing.T, envVarNames ...string) *fakes.CircleCI {
	t.Helper()

	fake := fakes.NewCircleCI(t)
	fake.AddProject(rocketSlug, "proj-rocket", "org-acme", "gh/acme")
	for _, name := range envVarNames {
		fake.AddProjectEnvVar(rocketSlug, name, "")
	}

	return fake
}

// openRocketConfig registers a config file belonging to acme/rocket, which is
// what env vars are cached against.
func openRocketConfig(t *testing.T, c *Cache) *File {
	t.Helper()

	cached := c.FileCache.SetFile(File{
		TextDocument: protocol.TextDocumentItem{URI: uri.URI("file:///rocket/.circleci/config.yml")},
		Project:      circleci.Project{Slug: rocketSlug},
	})

	return &cached
}

// cachedEnvVarNames is what a file's env var names ended up as in the cache.
func cachedEnvVarNames(cache *Cache, cachedFile *File) []string {
	return cache.FileCache.GetFile(cachedFile.TextDocument.URI).EnvVariables
}

func TestLoadProjectEnvVariables(t *testing.T) {
	t.Run("caches the names from every page", func(t *testing.T) {
		fake := projectFake(t, "AWS_REGION", "DEPLOY_KEY", "NPM_TOKEN", "SENTRY_DSN", "SLACK_WEBHOOK")
		fake.SetPageLimit("project/envvar", 2)

		c := New()
		cachedFile := openRocketConfig(t, c)

		err := c.LoadProjectEnvVariables(configFor(fake.URL()), cachedFile)
		assert.NilError(t, err)

		t.Run("in the order the API reported them", func(t *testing.T) {
			names := cachedEnvVarNames(c, cachedFile)
			assert.Check(t, cmp.DeepEqual(names, []string{
				"AWS_REGION", "DEPLOY_KEY", "NPM_TOKEN", "SENTRY_DSN", "SLACK_WEBHOOK",
			}))
		})

		// Three pages of two, the last one short: the fetch has to follow the
		// page tokens rather than stop at the first response.
		t.Run("following the page tokens to the end", func(t *testing.T) {
			requestCount := fake.RequestCount(http.MethodGet, "/api/v2/project/gh/acme/rocket/envvar")
			assert.Check(t, cmp.Equal(requestCount, 3))

			requests := fake.Requests()
			assert.Assert(t, cmp.Len(requests, 3))

			_, firstAsksForAPage := requests[0].Query["page-token"]
			assert.Check(t, !firstAsksForAPage, "the first page must be asked for without a token")

			assert.Check(t, requests[1].Query["page-token"] != "", "second page must ask for a cursor")
			assert.Check(t, requests[2].Query["page-token"] != requests[1].Query["page-token"],
				"third page must ask for the cursor the second page reported")
		})

		t.Run("authenticating with Circle-Token", func(t *testing.T) {
			requests := fake.Requests()
			assert.Assert(t, cmp.Len(requests, 3))
			assert.Check(t, cmp.Equal(requests[0].CircleToken, testToken))
		})
	})

	// A project with no env vars is a 200 with an empty list, which has to
	// leave the cache empty rather than look like a failure.
	t.Run("caches nothing for a project with no env vars", func(t *testing.T) {
		fake := projectFake(t)

		c := New()
		cachedFile := openRocketConfig(t, c)

		err := c.LoadProjectEnvVariables(configFor(fake.URL()), cachedFile)
		assert.NilError(t, err)

		names := cachedEnvVarNames(c, cachedFile)
		assert.Check(t, cmp.Len(names, 0))
	})

	// A token that cannot read the project's env vars used to yield zero env
	// vars silently, indistinguishable from a project that has none.
	t.Run("reports a rejected token", func(t *testing.T) {
		fake := projectFake(t, "AWS_REGION")
		fake.RequireToken("a-different-token")

		c := New()
		cachedFile := openRocketConfig(t, c)

		err := c.LoadProjectEnvVariables(configFor(fake.URL()), cachedFile)
		assert.Check(t, httpcl.HasStatusCode(err, 401), "got %v", err)

		names := cachedEnvVarNames(c, cachedFile)
		assert.Check(t, cmp.Len(names, 0))
	})

	t.Run("reports an unknown project", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)

		c := New()
		cachedFile := openRocketConfig(t, c)

		err := c.LoadProjectEnvVariables(configFor(fake.URL()), cachedFile)
		assert.Check(t, httpcl.HasStatusCode(err, 404), "got %v", err)
	})

	t.Run("reports a malformed body", func(t *testing.T) {
		fake := projectFake(t, "AWS_REGION")
		fake.SetBody(rocketEnvRoute, "{")

		c := New()
		cachedFile := openRocketConfig(t, c)

		err := c.LoadProjectEnvVariables(configFor(fake.URL()), cachedFile)
		assert.Check(t, cmp.ErrorContains(err, "decode response"))

		names := cachedEnvVarNames(c, cachedFile)
		assert.Check(t, cmp.Len(names, 0))
	})

	// Completion is better off with the names it did read than with none, so a
	// failure part way through a paginated read keeps the earlier pages.
	t.Run("keeps the pages it read when a later one fails", func(t *testing.T) {
		fake := projectFake(t, "AWS_REGION", "DEPLOY_KEY", "NPM_TOKEN", "SENTRY_DSN")
		fake.SetPageLimit("project/envvar", 2)
		fake.FailAfter(rocketEnvRoute, 1, http.StatusInternalServerError)

		c := New()
		cachedFile := openRocketConfig(t, c)

		err := c.LoadProjectEnvVariables(configFor(fake.URL()), cachedFile)
		assert.Check(t, httpcl.HasStatusCode(err, 500), "got %v", err)

		names := cachedEnvVarNames(c, cachedFile)
		assert.Check(t, cmp.DeepEqual(names, []string{"AWS_REGION", "DEPLOY_KEY"}))
	})

	t.Run("reports an unreachable host", func(t *testing.T) {
		fake := projectFake(t, "AWS_REGION")
		api := configFor(fake.URL())
		fake.Close()

		c := New()
		cachedFile := openRocketConfig(t, c)

		err := c.LoadProjectEnvVariables(api, cachedFile)
		assert.Check(t, err != nil, "a host that is not answering must be reported")
	})

	// A self-hosted URL comes from user settings, so it is not necessarily a
	// URL at all.
	t.Run("reports an unusable host URL", func(t *testing.T) {
		c := New()
		cachedFile := openRocketConfig(t, c)

		err := c.LoadProjectEnvVariables(configFor("not a url"), cachedFile)
		assert.Check(t, err != nil, "an unparseable host must be reported, not panic")
	})
}

func TestProject(t *testing.T) {
	const projectRoute = "/api/v2/project/gh/acme/rocket"

	t.Run("resolves a slug once for every caller", func(t *testing.T) {
		fake := projectFake(t)
		c := New()

		var wg sync.WaitGroup
		for range 10 {
			wg.Go(func() {
				project, err := c.Project(configFor(fake.URL()), rocketSlug)
				assert.Check(t, err)
				assert.Check(t, cmp.Equal(project.OrganizationId, "org-acme"))
			})
		}
		wg.Wait()

		assert.Check(t, cmp.Equal(fake.RequestCount(http.MethodGet, projectRoute), 1))
	})

	// A repository that is not a CircleCI project is asked about on every edit
	// of its config.
	t.Run("remembers that a slug names no project", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		c := New()

		for range 2 {
			project, err := c.Project(configFor(fake.URL()), rocketSlug)
			assert.NilError(t, err)
			assert.Check(t, cmp.DeepEqual(project, circleci.Project{}))
		}

		assert.Check(t, cmp.Equal(fake.RequestCount(http.MethodGet, projectRoute), 1))
	})

	t.Run("does not remember a failure", func(t *testing.T) {
		fake := projectFake(t)
		c := New()

		t.Run("fail the lookup", func(t *testing.T) {
			fake.SetStatus("GET "+projectRoute, http.StatusInternalServerError)
			_, err := c.Project(configFor(fake.URL()), rocketSlug)
			assert.Check(t, httpcl.HasStatusCode(err, 500), "got %v", err)
		})

		t.Run("check the next call resolves it", func(t *testing.T) {
			fake.SetStatus("GET "+projectRoute, 0)
			project, err := c.Project(configFor(fake.URL()), rocketSlug)
			assert.NilError(t, err)
			assert.Check(t, cmp.Equal(project.Slug, rocketSlug))
		})
	})
}
