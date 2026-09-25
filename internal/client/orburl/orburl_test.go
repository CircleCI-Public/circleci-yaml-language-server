package orburl_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/orburl"
)

func TestFetch(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/orb.yml":
			_, _ = w.Write([]byte("version: 2.1\n"))
		case "/forbidden.yml":
			w.WriteHeader(http.StatusForbidden)
		case "/broken.yml":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	cfg := orburl.Config{Transport: srv.Client().Transport}

	t.Run("an orb that is there is its source", func(t *testing.T) {
		source, found, err := orburl.Fetch(context.Background(), cfg, srv.URL+"/orb.yml")
		assert.Check(t, err)
		assert.Check(t, found)
		assert.Check(t, cmp.Equal(source, "version: 2.1\n"))
	})

	for _, name := range []string{"/missing.yml", "/forbidden.yml"} {
		t.Run("a host that won't serve "+name+" says it isn't there", func(t *testing.T) {
			_, found, err := orburl.Fetch(context.Background(), cfg, srv.URL+name)
			assert.Check(t, err)
			assert.Check(t, !found)
		})
	}

	t.Run("a host that fails is an error", func(t *testing.T) {
		_, found, err := orburl.Fetch(context.Background(), cfg, srv.URL+"/broken.yml")
		assert.Check(t, cmp.ErrorContains(err, "500"))
		assert.Check(t, !found)
	})

	t.Run("an orb not served over https isn't fetched", func(t *testing.T) {
		_, _, err := orburl.Fetch(context.Background(), cfg, "http://example.com/orb.yml")
		assert.Check(t, cmp.ErrorIs(err, orburl.ErrNotHTTPS))
	})
}

// gitHubFake stands in for every host: a transport sends each request to it,
// whatever host the URL names, and it records the Authorization header of
// each request it gets. It serves a public.yml to anyone, and a private.yml
// only to the token "secret"; anything else is 404, as GitHub answers for a
// file it won't show.
type gitHubFake struct {
	srv  *httptest.Server
	auth []string
}

func newGitHubFake(t *testing.T) *gitHubFake {
	t.Helper()

	fake := &gitHubFake{}
	fake.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fake.auth = append(fake.auth, r.Host+" "+r.Header.Get("Authorization"))
		switch {
		case strings.HasSuffix(r.URL.Path, "/public.yml"),
			strings.HasSuffix(r.URL.Path, "/private.yml") && r.Header.Get("Authorization") == "token secret":
			_, _ = w.Write([]byte("version: 2.1\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(fake.srv.Close)

	return fake
}

func (fake *gitHubFake) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.URL.Scheme = "http"
	r.URL.Host = fake.srv.Listener.Addr().String()
	return http.DefaultTransport.RoundTrip(r)
}

func TestFetchWithGitHubToken(t *testing.T) {
	const raw = "https://raw.githubusercontent.com/acme/orbs/main"

	fetch := func(t *testing.T, address string) (*gitHubFake, bool) {
		t.Helper()
		fake := newGitHubFake(t)
		cfg := orburl.Config{GitHubToken: "secret", Transport: fake}
		_, found, err := orburl.Fetch(context.Background(), cfg, address)
		assert.Check(t, err)
		return fake, found
	}

	t.Run("a public file is fetched without the token", func(t *testing.T) {
		fake, found := fetch(t, raw+"/public.yml")
		assert.Check(t, found)
		assert.Check(t, cmp.DeepEqual(fake.auth, []string{"raw.githubusercontent.com "}))
	})

	t.Run("a private file is fetched with the token once it isn't without", func(t *testing.T) {
		fake, found := fetch(t, raw+"/private.yml")
		assert.Check(t, found)
		assert.Check(t, cmp.DeepEqual(fake.auth, []string{
			"raw.githubusercontent.com ",
			"raw.githubusercontent.com token secret",
		}))
	})

	t.Run("the token is never sent to a host that isn't GitHub", func(t *testing.T) {
		fake, found := fetch(t, "https://example.com/private.yml")
		assert.Check(t, !found)
		assert.Check(t, cmp.DeepEqual(fake.auth, []string{"example.com "}))
	})
}

func TestNeedsGitHubToken(t *testing.T) {
	const address = "https://raw.githubusercontent.com/acme/orbs/main/go.yml"

	assert.Check(t, orburl.NeedsGitHubToken(orburl.Config{}, address))
	assert.Check(t, !orburl.NeedsGitHubToken(orburl.Config{GitHubToken: "secret"}, address))
	assert.Check(t, !orburl.NeedsGitHubToken(orburl.Config{}, "https://example.com/go.yml"))
}
