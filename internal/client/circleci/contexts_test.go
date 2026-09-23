package circleci

import (
	"net/http"
	"testing"
	"time"

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

func Test_getContext(t *testing.T) {
	t.Run("lists the contexts of an organization", func(t *testing.T) {
		fake := contextFake(t)

		res, err := getContext(configFor(fake.URL()), acmeOrgID, "", false)
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

		res, err := getContext(configFor(fake.URL()), acmeOrgID, "", true)
		assert.NilError(t, err)
		assert.Assert(t, cmp.Len(res.Items, 2))

		var names []string
		for _, envVar := range res.Items[0].EnvironmentVariables {
			names = append(names, envVar.Variable)
		}
		assert.Check(t, cmp.DeepEqual(names, []string{"DEPLOY_KEY", "SLACK_WEBHOOK"}))

		requests := fake.Requests()
		assert.Assert(t, cmp.Len(requests, 1))
		assert.Check(t, cmp.Equal(requests[0].Query["include-env-vars"], "true"))
	})

	t.Run("reports a rejected token", func(t *testing.T) {
		fake := contextFake(t)
		fake.RequireToken("a-different-token")

		res, err := getContext(configFor(fake.URL()), acmeOrgID, "", false)
		assert.Check(t, httpcl.HasStatusCode(err, 401), "got %v", err)
		assert.Check(t, cmp.Nil(res))
	})

	t.Run("reports a failing host", func(t *testing.T) {
		fake := contextFake(t)
		fake.SetStatus(contextRoute, http.StatusInternalServerError)

		res, err := getContext(configFor(fake.URL()), acmeOrgID, "", false)
		assert.Check(t, httpcl.HasStatusCode(err, 500), "got %v", err)
		assert.Check(t, cmp.Nil(res))
	})

	t.Run("reports a malformed body", func(t *testing.T) {
		fake := contextFake(t)
		fake.SetBody(contextRoute, "{")

		res, err := getContext(configFor(fake.URL()), acmeOrgID, "", false)
		assert.Check(t, cmp.ErrorContains(err, "decode response"))
		assert.Check(t, cmp.Nil(res))
	})

	t.Run("reports an unreachable host", func(t *testing.T) {
		fake := contextFake(t)
		api := configFor(fake.URL())
		fake.Close()

		res, err := getContext(api, acmeOrgID, "", false)
		assert.Check(t, err != nil, "a host that is not answering must be reported")
		assert.Check(t, cmp.Nil(res))
	})
}
