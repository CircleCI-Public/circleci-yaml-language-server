package circleci_test

import (
	"net/http"
	"sync"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
)

const meRoute = "GET /api/v2/me"

// userFake builds a fake reporting one account. Account ids are memoised for
// the life of the process, so the memo is cleared around every case.
func userFake(t *testing.T) *fakes.CircleCI {
	t.Helper()

	circleci.ResetUserIds()
	t.Cleanup(circleci.ResetUserIds)

	fake := fakes.NewCircleCI(t)
	fake.SetUser("user-jane", "jane", "Jane Doe")

	return fake
}

func TestConfigGetUserId(t *testing.T) {
	t.Run("reports the account the token belongs to", func(t *testing.T) {
		fake := userFake(t)
		fake.RequireToken("the-real-token")

		apiContext := circleci.Config{Token: "the-real-token", HostUrl: fake.URL()}

		userID := apiContext.GetUserId()
		assert.Check(t, cmp.Equal(userID, "user-jane"))
	})

	// The id is read for telemetry on paths that run per request, so it has to
	// be fetched once rather than once per caller.
	t.Run("fetches the id once", func(t *testing.T) {
		fake := userFake(t)
		apiContext := circleci.Config{Token: "the-real-token", HostUrl: fake.URL()}

		first := apiContext.GetUserId()
		second := apiContext.GetUserId()

		assert.Check(t, cmp.Equal(first, "user-jane"))
		assert.Check(t, cmp.Equal(second, first))

		requestCount := fake.RequestCount(http.MethodGet, "/api/v2/me")
		assert.Check(t, cmp.Equal(requestCount, 1))
	})

	// session.Settings carries its Config by value, so the memo cannot live in the
	// struct: a copy has to answer from what an earlier copy fetched.
	t.Run("fetches the id once across copies", func(t *testing.T) {
		fake := userFake(t)
		api := circleci.Config{Token: "the-real-token", HostUrl: fake.URL()}

		fetched := api.GetUserId()
		assert.Check(t, cmp.Equal(fetched, "user-jane"))

		copied := api
		fromCopy := copied.GetUserId()
		assert.Check(t, cmp.Equal(fromCopy, "user-jane"))

		requestCount := fake.RequestCount(http.MethodGet, "/api/v2/me")
		assert.Check(t, cmp.Equal(requestCount, 1), "a copy must not re-fetch the id")
	})

	// Signing in, or switching to a self-hosted host, changes which account the
	// id belongs to, so the memo is per host and token rather than global.
	t.Run("fetches again for a different token", func(t *testing.T) {
		fake := userFake(t)

		first := circleci.Config{Token: "the-first-token", HostUrl: fake.URL()}
		assert.Check(t, cmp.Equal(first.GetUserId(), "user-jane"))

		second := circleci.Config{Token: "the-second-token", HostUrl: fake.URL()}
		assert.Check(t, cmp.Equal(second.GetUserId(), "user-jane"))

		requestCount := fake.RequestCount(http.MethodGet, "/api/v2/me")
		assert.Check(t, cmp.Equal(requestCount, 2))
	})

	// Requests are served on their own goroutines from one shared session.Settings, so
	// the memo has to be safe to reach concurrently. Under -race this fails if
	// it is not; how many fetches the eight callers cost is not asserted,
	// because no lock is held across the request.
	t.Run("is safe to call concurrently", func(t *testing.T) {
		fake := userFake(t)
		api := circleci.Config{Token: "the-real-token", HostUrl: fake.URL()}

		ids := make([]string, 8)

		var callers sync.WaitGroup
		for i := range ids {
			callers.Add(1)

			go func() {
				defer callers.Done()

				ids[i] = api.GetUserId()
			}()
		}
		callers.Wait()

		for caller, id := range ids {
			assert.Check(t, cmp.Equal(id, "user-jane"), "caller %d", caller)
		}
	})

	t.Run("reports no id for a token the host rejects", func(t *testing.T) {
		fake := userFake(t)
		fake.RequireToken("the-real-token")

		apiContext := circleci.Config{Token: "a-stale-token", HostUrl: fake.URL()}

		userID := apiContext.GetUserId()
		assert.Check(t, cmp.Equal(userID, ""))
	})

	// The status has to be read rather than inferred from the decoded body: a
	// rate limit answers with an id of its own, which is not an account's.
	t.Run("reports no id for an error body carrying an id", func(t *testing.T) {
		fake := userFake(t)
		fake.SetStatus(meRoute, http.StatusTooManyRequests)
		fake.SetBody(meRoute, `{"id":"ratelimit-6a1f","message":"Rate limit exceeded"}`)

		apiContext := circleci.Config{Token: "the-real-token", HostUrl: fake.URL()}

		userID := apiContext.GetUserId()
		assert.Check(t, cmp.Equal(userID, ""), "an error body's id is not the account's")
	})

	// Nothing is memoised on failure, so a call after the host recovers gets a
	// real answer instead of the empty one.
	t.Run("retries after a failed fetch", func(t *testing.T) {
		fake := userFake(t)
		fake.FailAfter(meRoute, 0, http.StatusInternalServerError)

		apiContext := circleci.Config{Token: "the-real-token", HostUrl: fake.URL()}

		first := apiContext.GetUserId()
		assert.Check(t, cmp.Equal(first, ""))

		second := apiContext.GetUserId()
		assert.Check(t, cmp.Equal(second, ""))

		requestCount := fake.RequestCount(http.MethodGet, "/api/v2/me")
		assert.Check(t, cmp.Equal(requestCount, 2), "a failure must not be memoised")
	})

	t.Run("reports no id when the host is unreachable", func(t *testing.T) {
		fake := userFake(t)
		hostUrl := fake.URL()
		fake.Close()

		apiContext := circleci.Config{Token: "the-real-token", HostUrl: hostUrl}

		userID := apiContext.GetUserId()
		assert.Check(t, cmp.Equal(userID, ""))
	})

	// A self-hosted URL comes from user settings, so it is not necessarily a
	// URL at all.
	t.Run("reports no id for an unusable host URL", func(t *testing.T) {
		apiContext := circleci.Config{Token: "the-real-token", HostUrl: "not a url"}

		userID := apiContext.GetUserId()
		assert.Check(t, cmp.Equal(userID, ""), "an unparseable host must report no id, not panic")
	})
}
