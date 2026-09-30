// Package noprod keeps a test binary off the production CircleCI API.
//
// Tests that want the behaviour the language server has on circleci.com use
// testHelpers.DefaultSettings, whose host is the real one, so any request they
// make reaches production. The code under test swallows most fetch failures —
// a missing machine catalog or orb only skips a check — so refusing the
// request alone would go unnoticed. Main therefore also fails the run, naming
// every request it refused, so a test that needs API data has to get it from
// fakes.CircleCI instead.
package noprod

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
)

// ErrProduction is the error a refused request fails with.
var ErrProduction = errors.New("tests must not call the production CircleCI API; use fakes.CircleCI")

// Main runs the package's tests with requests to production refused, and
// fails the run if any test made one. Call it from the package's TestMain.
func Main(m *testing.M) {
	guard := &transport{next: http.DefaultTransport}
	http.DefaultTransport = guard

	code := m.Run()

	if refused := guard.refused(); len(refused) > 0 {
		_, _ = fmt.Fprintf(os.Stderr, "%v; the tests requested:\n  %s\n", ErrProduction, strings.Join(refused, "\n  "))
		code = 1
	}

	os.Exit(code)
}

type transport struct {
	next http.RoundTripper

	mu       sync.Mutex
	requests map[string]bool
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !isProduction(req.URL.Hostname()) {
		return t.next.RoundTrip(req)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.requests == nil {
		t.requests = map[string]bool{}
	}
	t.requests[req.Method+" "+req.URL.Path] = true

	return nil, ErrProduction
}

// refused lists each distinct request refused, sorted.
func (t *transport) refused() []string {
	t.mu.Lock()
	defer t.mu.Unlock()

	var requests []string
	for request := range t.requests {
		requests = append(requests, request)
	}
	slices.Sort(requests)

	return requests
}

func isProduction(host string) bool {
	return host == "circleci.com" || strings.HasSuffix(host, ".circleci.com")
}
