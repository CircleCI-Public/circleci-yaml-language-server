// Package fakes provides a fake CircleCI V3 API server for testing.
//
// It is modeled on the fake in the circleci-cli repository
// (internal/testing/fakes), which cannot be imported here because it lives
// under that module's internal/ directory. The builder API deliberately mirrors
// it — AddNamespace, AddOrbPackage, AddOrbVersion, URL — so the two read alike.
//
// It differs from that fake in three ways, each because this repository's code
// depends on the behavior:
//
//   - filter[namespace_id] is read under its real bracketed name.
//   - Collections paginate with real opaque page[cursor] values, so the
//     cursor-following loop in utils.GetPaged is actually exercised.
//   - An orb package carries every version it has, not just the latest, which
//     is what the real API returns and what orb version resolution relies on.
//
// Requests are served anonymously by default because the language server
// resolves public orbs for users who have not logged in. Call RequireToken to
// turn on Bearer enforcement.
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

// CircleCI is a fake CircleCI V3 API server.
type CircleCI struct {
	server *httptest.Server

	mu sync.RWMutex

	namespacesByName map[string]string // name -> id

	orbPackages       map[string]Orb        // id -> package
	orbPackagesByName map[string]string     // "ns/name" -> id
	orbVersions       map[string]OrbVersion // id -> version
	orbVersionsByOrb  map[string][]string   // orb id -> version ids, insertion order
	sourceStatus      map[string]int        // version id -> status override for /source
	requiredToken     string                // when set, Bearer auth is enforced
	requests          []Request             // every request received, in order
	statusOverrides   map[string]int        // "METHOD /path" -> status to return
	bodyOverrides     map[string]string     // "METHOD /path" -> raw body to return
	pageLimits        map[string]int        // path -> items served per page
	namespaces        map[string]Namespace  // id -> namespace
	failAfter         map[string]failAfter  // "METHOD /path" -> deferred failure
	v3OrbRoutesGone   bool                  // when set, the V3 orb/namespace routes answer 404
	namespaceHasMore  bool                  // when set, GraphQL reports another page of orbs
}

// failAfter defers a failure until a route has been hit a number of times, so
// that a later page of a paginated read can fail while earlier ones succeed.
type failAfter struct {
	after  int
	status int
}

// Namespace is a stored registry namespace.
type Namespace struct {
	ID   string
	Name string
}

// Orb is a stored orb package.
type Orb struct {
	ID        string
	Name      string // fully qualified "namespace/orb"
	NsID      string
	NsName    string
	IsPrivate bool
	IsListed  bool
}

// OrbVersion is a stored orb version.
type OrbVersion struct {
	ID        string
	OrbID     string
	OrbName   string // fully qualified "namespace/orb"
	Version   string
	Source    string
	CreatedAt string
}

// Request is a request the fake received.
type Request struct {
	Method        string
	Path          string
	Query         map[string]string
	Authorization string
	UserID        string
	UserAgent     string
}

// NewCircleCI starts a fake CircleCI V3 API server and closes it on cleanup.
func NewCircleCI(t *testing.T) *CircleCI {
	t.Helper()

	fake := &CircleCI{
		namespaces:        map[string]Namespace{},
		namespacesByName:  map[string]string{},
		orbPackages:       map[string]Orb{},
		orbPackagesByName: map[string]string{},
		orbVersions:       map[string]OrbVersion{},
		orbVersionsByOrb:  map[string][]string{},
		sourceStatus:      map[string]int{},
		statusOverrides:   map[string]int{},
		bodyOverrides:     map[string]string{},
		pageLimits:        map[string]int{},
		failAfter:         map[string]failAfter{},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/namespaces", fake.handleGetNamespace)
	mux.HandleFunc("GET /api/v3/orb/packages", fake.handleListOrbPackages)
	mux.HandleFunc("GET /api/v3/orb/versions", fake.handleListOrbVersions)
	mux.HandleFunc("GET /api/v3/orb/versions/{id}", fake.handleGetOrbVersion)
	mux.HandleFunc("GET /api/v3/orb/versions/{id}/source", fake.handleGetOrbVersionSource)
	mux.HandleFunc("POST /graphql-unstable", fake.handleGraphQL)

	fake.server = httptest.NewServer(fake.middleware(mux))
	t.Cleanup(fake.server.Close)

	return fake
}

// URL is the base URL of the fake, suitable for an ApiContext HostUrl.
func (f *CircleCI) URL() string {
	return f.server.URL
}

// --- Builder API ---

// AddNamespace registers a registry namespace.
func (f *CircleCI) AddNamespace(id, name string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.namespaces[id] = Namespace{ID: id, Name: name}
	f.namespacesByName[name] = id
}

// AddOrbPackage registers an orb package. orbName is the bare name; the fully
// qualified name is derived from nsName, matching how the API reports it.
func (f *CircleCI) AddOrbPackage(id, nsID, nsName, orbName string, isPrivate, isListed bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	fullName := nsName + "/" + orbName
	f.orbPackages[id] = Orb{
		ID:        id,
		Name:      fullName,
		NsID:      nsID,
		NsName:    nsName,
		IsPrivate: isPrivate,
		IsListed:  isListed,
	}
	f.orbPackagesByName[fullName] = id
}

// AddOrbVersion registers a version of an orb package. createdAt may be empty,
// in which case a fixed timestamp is used.
func (f *CircleCI) AddOrbVersion(id, orbID, orbName, version, source, createdAt string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if createdAt == "" {
		createdAt = "2026-01-15T10:30:00.000Z"
	}

	f.orbVersions[id] = OrbVersion{
		ID:        id,
		OrbID:     orbID,
		OrbName:   orbName,
		Version:   version,
		Source:    source,
		CreatedAt: createdAt,
	}
	f.orbVersionsByOrb[orbID] = append(f.orbVersionsByOrb[orbID], id)
}

// RequireToken turns on Bearer enforcement: every request must carry
// Authorization: Bearer <token> or be rejected with 401.
func (f *CircleCI) RequireToken(token string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.requiredToken = token
}

// SeedGoOrb loads the circleci namespace and a circleci/go orb with a spread of
// versions: several releases across two major versions, plus a development tag,
// which the API resolves by reference but leaves out of a package's version
// list. It is the shared fixture for orb resolution tests.
//
// The version ids are "ver-<version with dots and colons replaced by dashes>",
// so circleci/go@1.7.1 is "ver-1-7-1" and dev:alpha is "ver-dev-alpha".
func (f *CircleCI) SeedGoOrb() {
	f.AddNamespace("ns-circleci", "circleci")
	f.AddOrbPackage("orb-go", "ns-circleci", "circleci", "go", false, true)

	for _, version := range []string{
		"0.1.0",
		"1.7.0",
		"1.7.1",
		"1.7.3",
		"1.12.0",
		"dev:alpha",
		"4.0.0",
	} {
		id := "ver-" + strings.NewReplacer(".", "-", ":", "-").Replace(version)
		f.AddOrbVersion(id, "orb-go", "circleci/go", version, "# source of "+version+"\n", "")
	}
}

// SetPageLimit makes the collection at path serve at most n items per page, so
// that a caller's cursor-following can be exercised. path is the route below
// /api/v3, for example "orb/packages".
func (f *CircleCI) SetPageLimit(path string, n int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.pageLimits[path] = n
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

// SetSourceStatus makes the /source route for one orb version answer with
// status instead of its YAML.
func (f *CircleCI) SetSourceStatus(orbVersionID string, status int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.sourceStatus[orbVersionID] = status
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

// DisableV3OrbRoutes makes every V3 orb and namespace route answer 404, the way
// a CircleCI Server instance does. GraphQL keeps working, so this is the switch
// that exercises the fallback.
func (f *CircleCI) DisableV3OrbRoutes() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.v3OrbRoutesGone = true
}

// SetNamespaceHasMoreOrbs makes the GraphQL namespace query report a further
// page of orbs, which the query has no way to fetch.
func (f *CircleCI) SetNamespaceHasMoreOrbs() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.namespaceHasMore = true
}

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
			UserID:        r.Header.Get("user_id"),
			UserAgent:     r.Header.Get("User-Agent"),
		})
		route := r.Method + " " + r.URL.Path
		status := f.statusOverrides[route]
		body, hasBody := f.bodyOverrides[route]
		required := f.requiredToken
		orbRoutesGone := f.v3OrbRoutesGone
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

		// The V3 client sends "Bearer <token>"; the GraphQL client sends the
		// token raw. Both are accepted, as the real edge accepts both.
		authorization := r.Header.Get("Authorization")
		if required != "" && authorization != "Bearer "+required && authorization != required {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", "")

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
			writeError(w, status, "", http.StatusText(status), "")

			return
		}

		next.ServeHTTP(w, r)
	})
}

// --- Handlers ---

// isV3OrbRoute reports whether a path is one of the V3 orb or namespace routes
// that CircleCI Server does not serve.
func isV3OrbRoute(path string) bool {
	return path == "/api/v3/namespaces" || strings.HasPrefix(path, "/api/v3/orb/")
}

func (f *CircleCI) handleGetNamespace(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("filter[name]")
	if name == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "Bad Request.", "filter[name] is required")

		return
	}

	f.mu.RLock()
	id, ok := f.namespacesByName[name]
	f.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, "", "Not Found.", "")

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id":         id,
			"attributes": map[string]any{"name": name},
		},
	})
}

func (f *CircleCI) handleListOrbPackages(w http.ResponseWriter, r *http.Request) {
	nameFilter := r.URL.Query().Get("filter[name]")
	nsFilter := r.URL.Query().Get("filter[namespace_id]")

	f.mu.RLock()
	matched := []Orb{}
	for _, name := range f.sortedOrbNamesLocked() {
		orb := f.orbPackages[f.orbPackagesByName[name]]
		if nameFilter != "" && orb.Name != nameFilter {
			continue
		}
		if nsFilter != "" && orb.NsID != nsFilter {
			continue
		}
		matched = append(matched, orb)
	}

	entities := make([]any, 0, len(matched))
	for _, orb := range matched {
		entities = append(entities, f.orbEntityLocked(orb))
	}
	f.mu.RUnlock()

	f.writePage(w, r, "orb/packages", entities)
}

func (f *CircleCI) handleListOrbVersions(w http.ResponseWriter, r *http.Request) {
	ref := r.URL.Query().Get("filter[ref]")
	orbID := r.URL.Query().Get("filter[orb_id]")
	channel := r.URL.Query().Get("filter[channel]")

	f.mu.RLock()
	defer f.mu.RUnlock()

	// filter[ref] resolves a reference the way the real API does: an exact
	// version, a partial version ("1", "1.7"), "volatile", or a dev tag.
	if ref != "" {
		version, ok := f.resolveRefLocked(ref)
		if !ok {
			writeError(w, http.StatusNotFound, "", "Not Found.", "")

			return
		}
		if orbID != "" && version.OrbID != orbID {
			writeError(w, http.StatusNotFound, "", "Not Found.", "")

			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"data": []any{orbVersionEntity(version, false)},
			"page": map[string]any{"next": nil, "prev": nil},
		})

		return
	}

	entities := []any{}
	for _, id := range f.orbVersionsByOrb[orbID] {
		version := f.orbVersions[id]
		isDev := strings.HasPrefix(version.Version, "dev:")
		if channel == "stable" && isDev {
			continue
		}
		if channel == "dev" && !isDev {
			continue
		}
		entities = append(entities, orbVersionEntity(version, false))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": entities,
		"page": map[string]any{"next": nil, "prev": nil},
	})
}

func (f *CircleCI) handleGetOrbVersion(w http.ResponseWriter, r *http.Request) {
	f.mu.RLock()
	version, ok := f.orbVersions[r.PathValue("id")]
	f.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, "", "Not Found.", "")

		return
	}

	// include is single-valued: only the exact value "source" adds the source,
	// and anything else is ignored rather than rejected.
	includeSource := r.URL.Query().Get("include") == "source"

	writeJSON(w, http.StatusOK, map[string]any{"data": orbVersionEntity(version, includeSource)})
}

func (f *CircleCI) handleGetOrbVersionSource(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	f.mu.RLock()
	version, ok := f.orbVersions[id]
	status := f.sourceStatus[id]
	f.mu.RUnlock()

	if status != 0 {
		writeError(w, status, "", http.StatusText(status), "")

		return
	}

	if !ok {
		writeError(w, http.StatusNotFound, "", "Not Found.", "")

		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(version.Source))
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

// orbEntityLocked renders an orb package with every version it has, newest
// first, excluding dev tags. Callers must hold the read lock.
func (f *CircleCI) orbEntityLocked(orb Orb) map[string]any {
	versions := []any{}
	ids := f.orbVersionsByOrb[orb.ID]
	for i := len(ids) - 1; i >= 0; i-- {
		version := f.orbVersions[ids[i]]
		if strings.HasPrefix(version.Version, "dev:") {
			continue
		}
		versions = append(versions, map[string]any{
			"id": version.ID,
			"attributes": map[string]any{
				"version":    version.Version,
				"created_at": version.CreatedAt,
			},
		})
	}

	references := map[string]any{
		"namespace": map[string]any{"id": orb.NsID},
	}
	if len(versions) > 0 {
		references["orb_versions"] = versions
	}

	return map[string]any{
		"id": orb.ID,
		"attributes": map[string]any{
			"name":                       orb.Name,
			"is_private":                 orb.IsPrivate,
			"is_listed":                  orb.IsListed,
			"last_30_days_build_count":   0,
			"last_30_days_project_count": 0,
			"last_30_days_org_count":     0,
		},
		"references": references,
	}
}

func orbVersionEntity(version OrbVersion, includeSource bool) map[string]any {
	attributes := map[string]any{
		"version":    version.Version,
		"created_at": version.CreatedAt,
	}
	if includeSource {
		attributes["source"] = version.Source
	}

	return map[string]any{
		"id":         version.ID,
		"attributes": attributes,
		"references": map[string]any{
			"orb_package": map[string]any{"id": version.OrbID},
		},
	}
}

// resolveRefLocked resolves "ns/orb@something" the way the API does. Callers
// must hold the read lock.
func (f *CircleCI) resolveRefLocked(ref string) (OrbVersion, bool) {
	name, wanted, found := strings.Cut(ref, "@")
	if !found {
		return OrbVersion{}, false
	}

	orbID, ok := f.orbPackagesByName[name]
	if !ok {
		return OrbVersion{}, false
	}

	// Newest first, so the first match wins for volatile and partial refs.
	ids := f.orbVersionsByOrb[orbID]
	candidates := make([]OrbVersion, 0, len(ids))
	for i := len(ids) - 1; i >= 0; i-- {
		candidates = append(candidates, f.orbVersions[ids[i]])
	}

	for _, candidate := range candidates {
		if candidate.Version == wanted {
			return candidate, true
		}
	}

	if wanted == "volatile" {
		for _, candidate := range candidates {
			if !strings.HasPrefix(candidate.Version, "dev:") {
				return candidate, true
			}
		}

		return OrbVersion{}, false
	}

	// A partial version matches the newest release under that prefix.
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate.Version, wanted+".") {
			return candidate, true
		}
	}

	return OrbVersion{}, false
}

func (f *CircleCI) sortedOrbNamesLocked() []string {
	names := make([]string, 0, len(f.orbPackagesByName))
	for name := range f.orbPackagesByName {
		names = append(names, name)
	}

	// Insertion order is not meaningful for a map, and the real API returns a
	// stable ordering, so sort to keep pagination deterministic.
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && names[j] < names[j-1]; j-- {
			names[j], names[j-1] = names[j-1], names[j]
		}
	}

	return names
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
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

// --- GraphQL ---

// handleGraphQL serves the orb and namespace queries the language server sends
// when a host has no V3 orb routes, from the same stored state the V3 handlers
// use.
//
// It does not parse GraphQL. It matches on which root field a query selects,
// which is enough for the fixed set of queries in pkg/utils/orbregistry.go, and
// reproduces the behaviour that matters: a missing orb, version or namespace
// comes back as a null member of data rather than as an error.
func (f *CircleCI) handleGraphQL(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"errors": []any{map[string]any{"message": "could not decode request"}},
		})

		return
	}

	variable := func(name string) string {
		value, _ := body.Variables[name].(string)

		return value
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	// orbVersion is checked before orb: "orbVersion(" does not contain "orb(".
	switch {
	case strings.Contains(body.Query, "orbVersion("):
		f.graphQLOrbVersionLocked(w, variable("orbVersionRef"))
	case strings.Contains(body.Query, "registryNamespace("):
		f.graphQLRegistryNamespaceLocked(w, variable("name"), strings.Contains(body.Query, "orbs("))
	case strings.Contains(body.Query, "orb("):
		f.graphQLOrbLocked(w, variable("orbName"))
	default:
		writeJSON(w, http.StatusOK, map[string]any{
			"errors": []any{map[string]any{"message": "unrecognised query"}},
		})
	}
}

func (f *CircleCI) graphQLOrbLocked(w http.ResponseWriter, name string) {
	id, ok := f.orbPackagesByName[name]
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"orb": nil}})

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{"orb": f.graphQLOrbNodeLocked(f.orbPackages[id])},
	})
}

func (f *CircleCI) graphQLOrbVersionLocked(w http.ResponseWriter, ref string) {
	version, ok := f.resolveRefLocked(ref)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"orbVersion": nil}})

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"orbVersion": map[string]any{
				"id":      version.ID,
				"version": version.Version,
				"source":  version.Source,
				"orb":     f.graphQLOrbNodeLocked(f.orbPackages[version.OrbID]),
			},
		},
	})
}

func (f *CircleCI) graphQLRegistryNamespaceLocked(w http.ResponseWriter, name string, withOrbs bool) {
	id, ok := f.namespacesByName[name]
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"registryNamespace": nil}})

		return
	}

	namespace := map[string]any{"id": id, "name": name}

	if withOrbs {
		edges := []any{}
		for _, orbName := range f.sortedOrbNamesLocked() {
			orb := f.orbPackages[f.orbPackagesByName[orbName]]
			if orb.NsID != id {
				continue
			}
			edges = append(edges, map[string]any{
				"cursor": orb.ID,
				"node":   f.graphQLOrbNodeLocked(orb),
			})
		}
		// A namespace claiming another page has to report a totalCount beyond
		// what it served, or the two contradict each other.
		totalCount := len(edges)
		if f.namespaceHasMore {
			totalCount++
		}
		namespace["orbs"] = map[string]any{
			"totalCount": totalCount,
			"pageInfo":   map[string]any{"hasNextPage": f.namespaceHasMore},
			"edges":      edges,
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{"registryNamespace": namespace},
	})
}

// graphQLOrbNodeLocked renders an orb with its released versions, newest first
// and excluding development tags, matching what versions(count:) returns.
func (f *CircleCI) graphQLOrbNodeLocked(orb Orb) map[string]any {
	versions := []any{}
	ids := f.orbVersionsByOrb[orb.ID]
	for i := len(ids) - 1; i >= 0; i-- {
		version := f.orbVersions[ids[i]]
		if strings.HasPrefix(version.Version, "dev:") {
			continue
		}
		versions = append(versions, map[string]any{"version": version.Version})
	}

	return map[string]any{
		"id":       orb.ID,
		"name":     orb.Name,
		"versions": versions,
	}
}
