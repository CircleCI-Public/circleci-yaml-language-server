package methods

// These tests drive the method layer directly rather than over a jsonrpc2
// connection: updateProjectEnvVariables reads only the cache and the language
// server context, so the acceptance harness is not needed to cover it.

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/logging"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
)

const (
	rocketSlug     = "gh/acme/rocket"
	rocketEnvRoute = "GET /api/v2/project/gh/acme/rocket/envvar"
	rocketURI      = uri.URI("file:///rocket/.circleci/config.yml")
)

// rocketMethods builds a Methods over a fake serving acme/rocket with the env
// vars named, and a cache holding one open config file of that project.
//
// The client is left nil: nothing on this path touches it, and a nil
// client is a louder failure than a stub if that ever stops being true.
func rocketMethods(t *testing.T, token string, envVarNames ...string) (*Methods, *fakes.CircleCI) {
	t.Helper()

	fake := fakes.NewCircleCI(t)
	fake.AddProject(rocketSlug, "proj-rocket", "org-acme", "gh/acme")
	for _, name := range envVarNames {
		fake.AddProjectEnvVar(rocketSlug, name, "")
	}

	methods := New(context.Background(), nil, cache.New(), session.Settings{
		Api: circleci.Config{Token: token, HostUrl: fake.URL()},
	}, "")

	return methods, fake
}

// openRocketConfig caches a config file of acme/rocket, carrying the env var
// names given as already known.
func openRocketConfig(t *testing.T, methods *Methods, known ...string) *cache.File {
	t.Helper()

	cached := methods.Cache.FileCache.SetFile(cache.File{
		TextDocument: protocol.TextDocumentItem{URI: rocketURI},
		Project:      circleci.Project{Slug: rocketSlug},
		EnvVariables: known,
	})

	return &cached
}

// captureLog collects what the method layer logs during a case.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()

	logged := &bytes.Buffer{}
	previous := slog.Default()
	slog.SetDefault(logging.New(logged, true))
	t.Cleanup(func() { slog.SetDefault(previous) })

	return logged
}

func Test_updateProjectEnvVariables(t *testing.T) {
	t.Run("caches the project's env var names", func(t *testing.T) {
		methods, _ := rocketMethods(t, "a-token", "AWS_REGION", "DEPLOY_KEY")
		cachedFile := openRocketConfig(t, methods)

		methods.updateProjectEnvVariables(cachedFile)

		file := methods.Cache.FileCache.GetFile(rocketURI)
		assert.Assert(t, file != nil)
		assert.Check(t, cmp.DeepEqual(file.EnvVariables, []string{"AWS_REGION", "DEPLOY_KEY"}))
	})

	// A variable deleted in the CircleCI UI has to stop being offered, so the
	// refetch replaces what was cached rather than adding to it.
	t.Run("drops names the project no longer has", func(t *testing.T) {
		methods, _ := rocketMethods(t, "a-token", "AWS_REGION")
		cachedFile := openRocketConfig(t, methods, "AWS_REGION", "RETIRED_KEY")

		methods.updateProjectEnvVariables(cachedFile)

		file := methods.Cache.FileCache.GetFile(rocketURI)
		assert.Assert(t, file != nil)
		assert.Check(t, cmp.DeepEqual(file.EnvVariables, []string{"AWS_REGION"}))
	})

	// Without a token the route answers for nobody, so it is not worth asking:
	// the cached names are cleared and the API is left alone.
	t.Run("does not call the API without a token", func(t *testing.T) {
		methods, fake := rocketMethods(t, "", "AWS_REGION")
		cachedFile := openRocketConfig(t, methods, "STALE_KEY")

		methods.updateProjectEnvVariables(cachedFile)

		file := methods.Cache.FileCache.GetFile(rocketURI)
		assert.Assert(t, file != nil)
		assert.Check(t, cmp.Len(file.EnvVariables, 0))

		requestCount := fake.RequestCount(http.MethodGet, "/api/v2/project/gh/acme/rocket/envvar")
		assert.Check(t, cmp.Equal(requestCount, 0))
	})

	// A failing fetch degrades to offering nothing. It must not take the
	// document's diagnostics down with it, so the error is logged and the
	// method returns.
	t.Run("logs a failing fetch and leaves the file cached", func(t *testing.T) {
		methods, fake := rocketMethods(t, "a-token", "AWS_REGION")
		fake.SetStatus(rocketEnvRoute, http.StatusInternalServerError)
		cachedFile := openRocketConfig(t, methods, "STALE_KEY")
		logged := captureLog(t)

		methods.updateProjectEnvVariables(cachedFile)

		file := methods.Cache.FileCache.GetFile(rocketURI)
		assert.Assert(t, file != nil, "the file must stay cached when its env vars cannot be read")
		assert.Check(t, cmp.Len(file.EnvVariables, 0))

		assert.Check(t, cmp.Contains(logged.String(), "error getting project environment variables"))
		assert.Check(t, cmp.Contains(logged.String(), "500 Internal Server Error"))
	})

	// Completion is better off with the names it did read than with none.
	t.Run("keeps the pages it read when a later one fails", func(t *testing.T) {
		methods, fake := rocketMethods(t, "a-token", "AWS_REGION", "DEPLOY_KEY", "NPM_TOKEN")
		fake.SetPageLimit("project/envvar", 2)
		fake.FailAfter(rocketEnvRoute, 1, http.StatusInternalServerError)
		cachedFile := openRocketConfig(t, methods)
		logged := captureLog(t)

		methods.updateProjectEnvVariables(cachedFile)

		file := methods.Cache.FileCache.GetFile(rocketURI)
		assert.Assert(t, file != nil)
		assert.Check(t, cmp.DeepEqual(file.EnvVariables, []string{"AWS_REGION", "DEPLOY_KEY"}))

		assert.Check(t, cmp.Contains(logged.String(), "500 Internal Server Error"))
	})
}
