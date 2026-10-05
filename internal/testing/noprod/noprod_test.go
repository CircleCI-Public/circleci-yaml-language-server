package noprod

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestTransport(t *testing.T) {
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(local.Close)

	guard := &transport{next: http.DefaultTransport}
	client := &http.Client{Transport: guard}

	t.Run("refuses production and its subdomains", func(t *testing.T) {
		for _, url := range []string{
			"https://circleci.com/api/v3/catalog/resource-classes",
			"https://circleci.com/api/v3/catalog/resource-classes?page=2",
			"https://app.circleci.com/api/v2/me",
		} {
			resp, err := client.Get(url)
			if resp != nil {
				_ = resp.Body.Close()
			}
			assert.Check(t, cmp.ErrorIs(err, ErrProduction), "request to %s", url)
		}
	})

	t.Run("lets other hosts through", func(t *testing.T) {
		resp, err := client.Get(local.URL + "/api/v3/catalog/resource-classes")
		assert.NilError(t, err)
		_ = resp.Body.Close()
		assert.Check(t, cmp.Equal(resp.StatusCode, http.StatusNoContent))
	})

	t.Run("does not mistake a lookalike host for production", func(t *testing.T) {
		for _, host := range []string{"notcircleci.com", "circleci.com.example.org"} {
			assert.Check(t, !isProduction(host), "host %s", host)
		}
	})

	t.Run("lists each refused request once", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(guard.refused(), []string{
			"GET /api/v2/me",
			"GET /api/v3/catalog/resource-classes",
		}))
	})
}
