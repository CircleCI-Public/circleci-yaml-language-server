package circleci_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
)

// changingCredentials is a CredentialSource whose credentials can be changed
// between requests, the way a session's are when the user logs in.
type changingCredentials struct {
	mu          sync.Mutex
	credentials circleci.Credentials
}

func (source *changingCredentials) Credentials() circleci.Credentials {
	source.mu.Lock()
	defer source.mu.Unlock()

	return source.credentials
}

func (source *changingCredentials) set(credentials circleci.Credentials) {
	source.mu.Lock()
	defer source.mu.Unlock()

	source.credentials = credentials
}

func TestClientCredentials(t *testing.T) {
	t.Run("one client follows its credentials as they change", func(t *testing.T) {
		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-1", "circleci")
		source := &changingCredentials{credentials: circleci.Credentials{HostURL: fake.URL()}}
		client := circleci.NewClient(source, false)

		get := func(t *testing.T) {
			t.Helper()
			_, err := circleci.FetchNamespace(t.Context(), client, "circleci")
			assert.NilError(t, err)
		}

		t.Run("ask anonymously", get)
		t.Run("log in and ask again", func(t *testing.T) {
			source.set(circleci.Credentials{HostURL: fake.URL(), Token: "new-token", UserID: "user-1"})
			get(t)
		})
		t.Run("log out and ask again", func(t *testing.T) {
			source.set(circleci.Credentials{HostURL: fake.URL()})
			get(t)
		})

		t.Run("check each request carried the credentials of its moment", func(t *testing.T) {
			requests := fake.Requests()
			assert.Assert(t, cmp.Len(requests, 3))

			assert.Check(t, cmp.Equal(requests[0].Authorization, ""))
			assert.Check(t, cmp.Equal(requests[0].UserID, ""))
			assert.Check(t, cmp.Equal(requests[1].Authorization, "Bearer new-token"))
			assert.Check(t, cmp.Equal(requests[1].UserID, "user-1"))
			assert.Check(t, cmp.Equal(requests[2].Authorization, ""))
			assert.Check(t, cmp.Equal(requests[2].UserID, ""))
		})
	})

	t.Run("one client follows its host as it changes", func(t *testing.T) {
		first := fakes.NewCircleCI(t)
		first.AddNamespace("ns-1", "circleci")
		second := fakes.NewCircleCI(t)
		second.AddNamespace("ns-2", "circleci")
		source := &changingCredentials{credentials: circleci.Credentials{HostURL: first.URL(), Token: "first-token"}}
		client := circleci.NewClient(source, false)

		_, err := circleci.FetchNamespace(t.Context(), client, "circleci")
		assert.NilError(t, err)

		source.set(circleci.Credentials{HostURL: second.URL(), Token: "second-token"})
		namespace, err := circleci.FetchNamespace(t.Context(), client, "circleci")
		assert.NilError(t, err)
		assert.Check(t, cmp.Equal(namespace.ID, "ns-2"))

		toFirst := first.Requests()
		assert.Assert(t, cmp.Len(toFirst, 1))
		assert.Check(t, cmp.Equal(toFirst[0].Authorization, "Bearer first-token"))
		toSecond := second.Requests()
		assert.Assert(t, cmp.Len(toSecond, 1))
		assert.Check(t, cmp.Equal(toSecond[0].Authorization, "Bearer second-token"))
	})

	// The credentials are added by the transport, which sees every hop of a
	// redirect, so it has to be the one to keep them to their own host.
	t.Run("keeps the credentials from a host a redirect leads to", func(t *testing.T) {
		var elsewhere http.Header
		other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			elsewhere = r.Header.Clone()
			_, _ = w.Write([]byte(`{"data": {}}`))
		}))
		t.Cleanup(other.Close)

		redirecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, other.URL+r.URL.Path, http.StatusFound)
		}))
		t.Cleanup(redirecting.Close)

		client := circleci.NewClient(circleci.Credentials{
			HostURL: redirecting.URL,
			Token:   "secret-token",
			UserID:  "user-1",
		}, false)

		var data struct{}
		err := client.Get(t.Context(), "namespaces", nil, &data)
		assert.NilError(t, err)

		assert.Assert(t, elsewhere != nil, "the redirect must have been followed")
		authorization := elsewhere.Get("Authorization")
		assert.Check(t, cmp.Equal(authorization, ""))
		userID := elsewhere.Get("User_id")
		assert.Check(t, cmp.Equal(userID, ""))
	})
}
