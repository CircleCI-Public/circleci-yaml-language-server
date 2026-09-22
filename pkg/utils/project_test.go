package utils_test

import (
	"net/http"
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/testHelpers"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
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
func openRocketConfig(t *testing.T, cache *utils.Cache) *utils.CachedFile {
	t.Helper()

	cached := cache.FileCache.SetFile(utils.CachedFile{
		TextDocument: protocol.TextDocumentItem{URI: protocol.URI("file:///rocket/.circleci/config.yml")},
		Project:      utils.Project{Slug: rocketSlug},
	})

	return &cached
}

// cachedEnvVarNames is what a file's env var names ended up as in the cache.
func cachedEnvVarNames(cache *utils.Cache, cachedFile *utils.CachedFile) []string {
	return cache.FileCache.GetFile(cachedFile.TextDocument.URI).EnvVariables
}

func TestGetAllProjectEnvVariables(t *testing.T) {
	t.Run("caches the names from every page", func(t *testing.T) {
		fake := projectFake(t, "AWS_REGION", "DEPLOY_KEY", "NPM_TOKEN", "SENTRY_DSN", "SLACK_WEBHOOK")
		fake.SetPageLimit("project/envvar", 2)

		cache := utils.CreateCache()
		cachedFile := openRocketConfig(t, cache)

		err := utils.GetAllProjectEnvVariables(testHelpers.GetLsContextForHost(fake.URL()), cache, cachedFile)
		assert.NilError(t, err)

		t.Run("in the order the API reported them", func(t *testing.T) {
			names := cachedEnvVarNames(cache, cachedFile)
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
			assert.Check(t, cmp.Equal(requests[0].CircleToken, "XXXXXXXXXXXX"))
		})
	})

	// A project with no env vars is a 200 with an empty list, which has to
	// leave the cache empty rather than look like a failure.
	t.Run("caches nothing for a project with no env vars", func(t *testing.T) {
		fake := projectFake(t)

		cache := utils.CreateCache()
		cachedFile := openRocketConfig(t, cache)

		err := utils.GetAllProjectEnvVariables(testHelpers.GetLsContextForHost(fake.URL()), cache, cachedFile)
		assert.NilError(t, err)

		names := cachedEnvVarNames(cache, cachedFile)
		assert.Check(t, cmp.Len(names, 0))
	})

	// A token that cannot read the project's env vars used to yield zero env
	// vars silently, indistinguishable from a project that has none.
	t.Run("reports a rejected token", func(t *testing.T) {
		fake := projectFake(t, "AWS_REGION")
		fake.RequireToken("a-different-token")

		cache := utils.CreateCache()
		cachedFile := openRocketConfig(t, cache)

		err := utils.GetAllProjectEnvVariables(testHelpers.GetLsContextForHost(fake.URL()), cache, cachedFile)
		assert.Check(t, cmp.ErrorContains(err, "HTTP 401"))

		names := cachedEnvVarNames(cache, cachedFile)
		assert.Check(t, cmp.Len(names, 0))
	})

	t.Run("reports an unknown project", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)

		cache := utils.CreateCache()
		cachedFile := openRocketConfig(t, cache)

		err := utils.GetAllProjectEnvVariables(testHelpers.GetLsContextForHost(fake.URL()), cache, cachedFile)
		assert.Check(t, cmp.ErrorContains(err, "HTTP 404"))
	})

	t.Run("reports a malformed body", func(t *testing.T) {
		fake := projectFake(t, "AWS_REGION")
		fake.SetBody(rocketEnvRoute, "{")

		cache := utils.CreateCache()
		cachedFile := openRocketConfig(t, cache)

		err := utils.GetAllProjectEnvVariables(testHelpers.GetLsContextForHost(fake.URL()), cache, cachedFile)
		assert.Check(t, cmp.ErrorContains(err, "unexpected end of JSON input"))

		names := cachedEnvVarNames(cache, cachedFile)
		assert.Check(t, cmp.Len(names, 0))
	})

	// Completion is better off with the names it did read than with none, so a
	// failure part way through a paginated read keeps the earlier pages.
	t.Run("keeps the pages it read when a later one fails", func(t *testing.T) {
		fake := projectFake(t, "AWS_REGION", "DEPLOY_KEY", "NPM_TOKEN", "SENTRY_DSN")
		fake.SetPageLimit("project/envvar", 2)
		fake.FailAfter(rocketEnvRoute, 1, http.StatusInternalServerError)

		cache := utils.CreateCache()
		cachedFile := openRocketConfig(t, cache)

		err := utils.GetAllProjectEnvVariables(testHelpers.GetLsContextForHost(fake.URL()), cache, cachedFile)
		assert.Check(t, cmp.ErrorContains(err, "HTTP 500"))

		names := cachedEnvVarNames(cache, cachedFile)
		assert.Check(t, cmp.DeepEqual(names, []string{"AWS_REGION", "DEPLOY_KEY"}))
	})

	t.Run("reports an unreachable host", func(t *testing.T) {
		fake := projectFake(t, "AWS_REGION")
		lsContext := testHelpers.GetLsContextForHost(fake.URL())
		fake.Close()

		cache := utils.CreateCache()
		cachedFile := openRocketConfig(t, cache)

		err := utils.GetAllProjectEnvVariables(lsContext, cache, cachedFile)
		assert.Check(t, err != nil, "a host that is not answering must be reported")
	})

	// A self-hosted URL comes from user settings, so it is not necessarily a
	// URL at all.
	t.Run("reports an unusable host URL", func(t *testing.T) {
		cache := utils.CreateCache()
		cachedFile := openRocketConfig(t, cache)

		err := utils.GetAllProjectEnvVariables(testHelpers.GetLsContextForHost("not a url"), cache, cachedFile)
		assert.Check(t, err != nil, "an unparseable host must be reported, not panic")
	})
}
