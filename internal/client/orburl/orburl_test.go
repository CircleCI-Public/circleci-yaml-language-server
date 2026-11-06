package orburl_test

import (
	"context"
	"errors"
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
		source, found, err := orburl.Fetch(t.Context(), cfg, srv.URL+"/orb.yml")
		assert.Check(t, err)
		assert.Check(t, found)
		assert.Check(t, cmp.Equal(source, "version: 2.1\n"))
	})

	for _, name := range []string{"/missing.yml", "/forbidden.yml"} {
		t.Run("a host that won't serve "+name+" says it isn't there", func(t *testing.T) {
			_, found, err := orburl.Fetch(t.Context(), cfg, srv.URL+name)
			assert.Check(t, err)
			assert.Check(t, !found)
		})
	}

	t.Run("a host that fails is an error", func(t *testing.T) {
		_, found, err := orburl.Fetch(t.Context(), cfg, srv.URL+"/broken.yml")
		assert.Check(t, cmp.ErrorContains(err, "500"))
		assert.Check(t, !found)
	})

	t.Run("an orb not served over https isn't fetched", func(t *testing.T) {
		_, _, err := orburl.Fetch(t.Context(), cfg, "http://example.com/orb.yml")
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
		_, found, err := orburl.Fetch(t.Context(), cfg, address)
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

func TestFetchFromClient(t *testing.T) {
	const raw = "https://raw.githubusercontent.com/acme/orbs/main"

	// fromClient stands in for a client that fetches GitHub files: it serves
	// private.yml, fails for broken.yml, and records what it is asked.
	type fromClient struct{ asked []string }
	newConfig := func(t *testing.T, token string) (orburl.Config, *gitHubFake, *fromClient) {
		t.Helper()
		fake := newGitHubFake(t)
		client := &fromClient{}
		return orburl.Config{
			GitHubToken: token,
			Transport:   fake,
			FetchGitHubFile: func(_ context.Context, address string) (string, bool, error) {
				client.asked = append(client.asked, address)
				switch {
				case strings.HasSuffix(address, "/private.yml"):
					return "version: 2.1\n", true, nil
				case strings.HasSuffix(address, "/broken.yml"):
					return "", false, errors.New("the client failed")
				default:
					return "", false, nil
				}
			},
		}, fake, client
	}

	t.Run("a private file without a token is fetched by the client", func(t *testing.T) {
		cfg, fake, client := newConfig(t, "")
		source, found, err := orburl.Fetch(t.Context(), cfg, raw+"/private.yml")
		assert.Check(t, err)
		assert.Check(t, found)
		assert.Check(t, cmp.Equal(source, "version: 2.1\n"))
		assert.Check(t, cmp.DeepEqual(fake.auth, []string{"raw.githubusercontent.com "}))
		assert.Check(t, cmp.DeepEqual(client.asked, []string{raw + "/private.yml"}))
	})

	t.Run("a public file isn't asked of the client", func(t *testing.T) {
		cfg, _, client := newConfig(t, "")
		_, found, err := orburl.Fetch(t.Context(), cfg, raw+"/public.yml")
		assert.Check(t, err)
		assert.Check(t, found)
		assert.Check(t, cmp.Len(client.asked, 0))
	})

	t.Run("a token is used rather than the client", func(t *testing.T) {
		cfg, _, client := newConfig(t, "wrong")
		_, found, err := orburl.Fetch(t.Context(), cfg, raw+"/private.yml")
		assert.Check(t, err)
		assert.Check(t, !found, "the token can't see it, and its answer stands")
		assert.Check(t, cmp.Len(client.asked, 0))
	})

	t.Run("a file the client can't see isn't there", func(t *testing.T) {
		cfg, _, _ := newConfig(t, "")
		_, found, err := orburl.Fetch(t.Context(), cfg, raw+"/missing.yml")
		assert.Check(t, err)
		assert.Check(t, !found)
	})

	t.Run("the client failing is an error", func(t *testing.T) {
		cfg, _, _ := newConfig(t, "")
		_, found, err := orburl.Fetch(t.Context(), cfg, raw+"/broken.yml")
		assert.Check(t, cmp.ErrorContains(err, "the client failed"))
		assert.Check(t, !found)
	})

	t.Run("a file on a host that isn't GitHub is never asked of the client", func(t *testing.T) {
		cfg, _, client := newConfig(t, "")
		_, found, err := orburl.Fetch(t.Context(), cfg, "https://example.com/private.yml")
		assert.Check(t, err)
		assert.Check(t, !found)
		assert.Check(t, cmp.Len(client.asked, 0))
	})
}

func TestGitHubAt(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		_, _ = w.Write([]byte("version: 2.1\n"))
	}))
	t.Cleanup(srv.Close)

	transport, err := orburl.GitHubAt(srv.URL)
	assert.NilError(t, err)

	t.Run("a file on GitHub is fetched from the fake", func(t *testing.T) {
		_, found, err := orburl.Fetch(t.Context(), orburl.Config{Transport: transport}, "https://raw.githubusercontent.com/acme/orbs/main/go.yml")
		assert.Check(t, err)
		assert.Check(t, found)
		assert.Check(t, cmp.DeepEqual(paths, []string{"/acme/orbs/main/go.yml"}))
	})

	t.Run("empty is GitHub itself", func(t *testing.T) {
		transport, err := orburl.GitHubAt("")
		assert.Check(t, err)
		assert.Check(t, transport == nil)
	})
}

func TestNeedsGitHubToken(t *testing.T) {
	const address = "https://raw.githubusercontent.com/acme/orbs/main/go.yml"

	assert.Check(t, orburl.NeedsGitHubToken(orburl.Config{}, address))
	assert.Check(t, !orburl.NeedsGitHubToken(orburl.Config{GitHubToken: "secret"}, address))
	assert.Check(t, !orburl.NeedsGitHubToken(orburl.Config{}, "https://example.com/go.yml"))
}
