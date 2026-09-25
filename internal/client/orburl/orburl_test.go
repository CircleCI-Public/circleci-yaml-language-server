package orburl_test

import (
	"context"
	"net/http"
	"net/http/httptest"
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
