// Package fakes provides a fake CircleCI API server for testing.
//
// It is modeled on the fake in the circleci-cli repository
// (internal/testing/fakes), which cannot be imported here because it lives
// under that module's internal/ directory. The builder API deliberately mirrors
// it — AddNamespace, AddOrbPackage, AddOrbVersion, URL — so the two read alike.
//
// This file holds the server, the route table for the whole API, the request
// log and fault injection every route shares, and the rendering the handlers
// answer through. One file per domain holds that domain's state, builders and
// handlers: circleci_orbs.go, circleci_graphql.go, circleci_account.go,
// circleci_projects.go, circleci_contexts.go, circleci_catalog.go,
// circleci_runner.go and circleci_functions.go.
//
// Requests are served anonymously by default because the language server
// resolves public orbs for users who have not logged in. Call RequireToken to
// turn on token enforcement.
package fakes

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// CircleCI is a fake CircleCI API server.
type CircleCI struct {
	server *httptest.Server

	mu sync.RWMutex

	// Fault injection and the request log, shared by every route.
	requiredToken   string               // when set, token auth is enforced
	requests        []Request            // every request received, in order
	statusOverrides map[string]int       // "METHOD /path" -> status to return
	bodyOverrides   map[string]string    // "METHOD /path" -> raw body to return
	pageLimits      map[string]int       // collection -> items served per page
	failAfter       map[string]failAfter // "METHOD /path" -> deferred failure

	// One field per domain of the API. Each is guarded by the mutex above and
	// is served by the handlers in the file named against it.
	orbs      orbState       // circleci_orbs.go
	graphql   graphqlState   // circleci_graphql.go
	account   accountState   // circleci_account.go
	projects  projectsState  // circleci_projects.go
	contexts  contextsState  // circleci_contexts.go
	catalog   catalogState   // circleci_catalog.go
	runner    runnerState    // circleci_runner.go
	functions functionsState // circleci_functions.go
}

// failAfter defers a failure until a route has been hit a number of times, so
// that a later page of a paginated read can fail while earlier ones succeed.
type failAfter struct {
	after  int
	status int
}

// Request is a request the fake received.
type Request struct {
	Method        string
	Path          string
	Query         map[string]string
	Authorization string
	CircleToken   string
	UserID        string
	UserAgent     string
}

// NewCircleCI starts a fake CircleCI API server and closes it on cleanup.
func NewCircleCI(t testing.TB) *CircleCI {
	t.Helper()

	fake := &CircleCI{
		statusOverrides: map[string]int{},
		bodyOverrides:   map[string]string{},
		pageLimits:      map[string]int{},
		failAfter:       map[string]failAfter{},
		orbs:            newOrbState(),
		projects:        newProjectsState(),
		contexts:        newContextsState(),
		runner:          newRunnerState(),
	}

	// Every route the fake serves is registered here, so that the whole API
	// surface reads in one place. Each group's handlers live in the file named
	// against it.
	mux := http.NewServeMux()

	// The orb registry — circleci_orbs.go.
	mux.HandleFunc("GET /api/v3/namespaces", fake.handleGetNamespace)
	mux.HandleFunc("GET /api/v3/orb/packages", fake.handleListOrbPackages)
	mux.HandleFunc("GET /api/v3/orb/versions", fake.handleListOrbVersions)
	mux.HandleFunc("GET /api/v3/orb/versions/{id}", fake.handleGetOrbVersion)
	mux.HandleFunc("GET /api/v3/orb/versions/{id}/source", fake.handleGetOrbVersionSource)

	// The GraphQL fallback the registry uses on a host without the V3 orb
	// routes — circleci_graphql.go.
	mux.HandleFunc("POST /graphql-unstable", fake.handleGraphQL)

	// The signed-in account — circleci_account.go.
	mux.HandleFunc("GET /api/v2/me", fake.handleGetMe)

	// Projects and their environment variables — circleci_projects.go. A
	// project slug is three path segments ("gh/org/repo"), so it is matched
	// segment by segment rather than with a trailing wildcard, which would also
	// swallow the /envvar suffix.
	mux.HandleFunc("GET /api/v2/project/{vcs}/{org}/{project}", fake.handleGetProject)
	mux.HandleFunc("GET /api/v2/project/{vcs}/{org}/{project}/envvar", fake.handleGetProjectEnvVars)

	// Contexts and their environment variables — circleci_contexts.go.
	mux.HandleFunc("GET /api/v2/context", fake.handleListContexts)
	mux.HandleFunc("GET /api/v2/context/{id}/environment-variable", fake.handleListContextEnvVars)

	// The machine catalog — circleci_catalog.go.
	mux.HandleFunc("GET /api/v3/catalog/offerings", fake.handleGetOfferings)

	// The functions catalog — circleci_functions.go.
	mux.HandleFunc("GET /api/v3/function/packages", fake.handleListFunctionPackages)
	mux.HandleFunc("GET /api/v3/function/versions/{id}", fake.handleGetFunctionVersion)

	// Self-hosted runner resource classes — circleci_runner.go. The /api/v3
	// here is the runner service's own versioning, not the CircleCI V3 API
	// above it; the two share a prefix and nothing else.
	mux.HandleFunc("GET /api/v3/runner/resource", fake.handleListRunnerClasses)

	fake.server = httptest.NewServer(fake.middleware(mux))
	t.Cleanup(fake.server.Close)

	return fake
}

// URL is the base URL of the fake, suitable for a circleci.Config HostUrl.
func (f *CircleCI) URL() string {
	return f.server.URL
}

// Close stops the fake before the end of the test, so that a caller sees a host
// that has gone away rather than one answering an error. The cleanup closes it
// again, which is harmless.
func (f *CircleCI) Close() {
	f.server.Close()
}

// --- Fault injection ---

// RequireToken turns on token enforcement: every request must carry the token
// in one of the three headers the real edge accepts — Authorization: Bearer
// <token>, a raw Authorization, or Circle-Token — or be rejected with 401.
func (f *CircleCI) RequireToken(token string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.requiredToken = token
}

// SetPageLimit makes a collection serve at most n items per page, so that a
// caller's cursor-following can be exercised.
//
// collection names what to limit, and is one of "orb/packages",
// "project/envvar", "context" or "context/environment-variable".
func (f *CircleCI) SetPageLimit(collection string, n int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.pageLimits[collection] = n
}

// SetStatus makes every request to the given route answer with status. route is
// "METHOD /path", for example "GET /api/v3/orb/packages".
func (f *CircleCI) SetStatus(route string, status int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.statusOverrides[route] = status
}

// SetBody makes every request to the given route answer with body verbatim.
func (f *CircleCI) SetBody(route, body string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.bodyOverrides[route] = body
}

// FailAfter makes a route succeed for its first n requests and then answer with
// status. Use it to fail a later page of a paginated read: a plain SetStatus
// would fail the first page too, and a body override that always repeats the
// same cursor would never terminate.
func (f *CircleCI) FailAfter(route string, n, status int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.failAfter[route] = failAfter{after: n, status: status}
}

// --- Inspection ---

// Requests returns every request the fake received, in order.
func (f *CircleCI) Requests() []Request {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return append([]Request(nil), f.requests...)
}

// RequestCount returns how many requests the fake received for a route.
func (f *CircleCI) RequestCount(method, path string) int {
	f.mu.RLock()
	defer f.mu.RUnlock()

	count := 0
	for _, req := range f.requests {
		if req.Method == method && req.Path == path {
			count++
		}
	}

	return count
}

// --- Middleware ---

func (f *CircleCI) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := map[string]string{}
		for key := range r.URL.Query() {
			query[key] = r.URL.Query().Get(key)
		}

		f.mu.Lock()
		f.requests = append(f.requests, Request{
			Method:        r.Method,
			Path:          r.URL.Path,
			Query:         query,
			Authorization: r.Header.Get("Authorization"),
			CircleToken:   r.Header.Get("Circle-Token"),
			UserID:        r.Header.Get("user_id"),
			UserAgent:     r.Header.Get("User-Agent"),
		})
		route := r.Method + " " + r.URL.Path
		status := f.statusOverrides[route]
		body, hasBody := f.bodyOverrides[route]
		required := f.requiredToken
		orbRoutesGone := f.orbs.v3RoutesGone
		if deferred, ok := f.failAfter[route]; ok {
			hits := 0
			for _, seen := range f.requests {
				if seen.Method == r.Method && seen.Path == r.URL.Path {
					hits++
				}
			}
			if hits > deferred.after {
				status = deferred.status
				hasBody = false
			}
		}
		f.mu.Unlock()

		// A Server instance serves GraphQL but not the V3 orb routes.
		if orbRoutesGone && isV3OrbRoute(r.URL.Path) {
			writeError(w, http.StatusNotFound, "", "Not Found.", "")

			return
		}

		// The V3 client sends "Bearer <token>", the GraphQL client sends the
		// token raw, and the V2 callers send Circle-Token. All three are
		// accepted, as the real edge accepts all three.
		authorization := r.Header.Get("Authorization")
		circleToken := r.Header.Get("Circle-Token")
		if required != "" && authorization != "Bearer "+required && authorization != required && circleToken != required {
			writeStatus(w, r.URL.Path, http.StatusUnauthorized)

			return
		}

		if hasBody {
			if status == 0 {
				status = http.StatusOK
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))

			return
		}

		if status != 0 {
			writeStatus(w, r.URL.Path, status)

			return
		}

		next.ServeHTTP(w, r)
	})
}

// --- Rendering ---

// writePage serves entities as a paginated collection. The cursor is the index
// of the next item, base64-encoded so that callers cannot meaningfully
// construct or decode it, as with the real API.
func (f *CircleCI) writePage(w http.ResponseWriter, r *http.Request, path string, entities []any) {
	f.mu.RLock()
	limit := f.pageLimits[path]
	f.mu.RUnlock()

	if requested := r.URL.Query().Get("page[limit]"); requested != "" {
		parsed, err := strconv.Atoi(requested)
		if err != nil || parsed < 1 || parsed > 1000 {
			writeError(w, http.StatusBadRequest, "validation_error", "Invalid Page Limit",
				"Page limit must be at most 1000, got "+requested)

			return
		}
		if limit == 0 || parsed < limit {
			limit = parsed
		}
	}

	start := 0
	if cursor := r.URL.Query().Get("page[cursor]"); cursor != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			writeError(w, http.StatusBadRequest, "validation_error", "Invalid Cursor", "")

			return
		}
		start, err = strconv.Atoi(string(decoded))
		if err != nil || start < 0 || start > len(entities) {
			writeError(w, http.StatusBadRequest, "validation_error", "Invalid Cursor", "")

			return
		}
	}

	end := len(entities)
	if limit > 0 && start+limit < end {
		end = start + limit
	}

	var next any
	if end < len(entities) {
		next = base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(end)))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": entities[start:end],
		"page": map[string]any{"next": next, "prev": nil},
	})
}

// writeV2Page serves items as a V2 collection. V2 paginates with an opaque
// page-token in the query string and reports the next one in the body, rather
// than with V3's page[cursor] and page.next. The token is the index of the next
// item, base64-encoded so that a caller can only feed back what it was given,
// as with the real API.
func (f *CircleCI) writeV2Page(w http.ResponseWriter, r *http.Request, key string, items []any) {
	f.mu.RLock()
	limit := f.pageLimits[key]
	f.mu.RUnlock()

	start := 0
	if token := r.URL.Query().Get("page-token"); token != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(token)
		if err != nil {
			writeV2Error(w, http.StatusBadRequest, "Invalid page-token")

			return
		}
		start, err = strconv.Atoi(string(decoded))
		if err != nil || start < 0 || start > len(items) {
			writeV2Error(w, http.StatusBadRequest, "Invalid page-token")

			return
		}
	}

	end := len(items)
	if limit > 0 && start+limit < end {
		end = start + limit
	}

	var next any
	if end < len(items) {
		next = base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(end)))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":           items[start:end],
		"next_page_token": next,
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// writeStatus reports a bare status in whichever error shape the route uses:
// V2 answers with {"message": ...}, everything else with V3's {"error": {...}}.
func writeStatus(w http.ResponseWriter, path string, status int) {
	if strings.HasPrefix(path, "/api/v2/") {
		writeV2Error(w, status, http.StatusText(status))

		return
	}

	writeError(w, status, "", http.StatusText(status), "")
}

// writeV2Error renders the {"message": ...} body the V2 API reports errors as.
func writeV2Error(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"message": message})
}

func writeError(w http.ResponseWriter, status int, errType, title, detail string) {
	body := map[string]any{"id": "fake-request-id"}
	if errType != "" {
		body["type"] = errType
	}
	if title != "" {
		body["title"] = title
	}
	if detail != "" {
		body["detail"] = detail
	}

	writeJSON(w, status, map[string]any{"error": body})
}
