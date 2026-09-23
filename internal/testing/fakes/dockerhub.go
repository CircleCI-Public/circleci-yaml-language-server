package fakes

// This file serves the Docker Hub API the language server reads images and tags
// from: https://docs.docker.com/reference/api/hub/latest/.
//
// It is a separate fake from CircleCI rather than another domain of it, because
// it is a different service on a different host, reached through a base URL of
// its own.
//
// Two things about Docker Hub are reproduced here because this repository's
// code depends on them:
//
//   - A collection reports its successor as an absolute next URL, which a
//     caller follows verbatim, so the fake hands out URLs of its own address.
//   - A page is small by default, which is why a caller that wants more than a
//     handful of tags has to ask for a bigger page or follow the next URL.

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// defaultPageSize is what Docker Hub serves when a request does not ask for a
// size, and is small enough that anything reading a popular repository has to
// paginate.
const defaultPageSize = 10

// DockerHub is a fake Docker Hub API server.
type DockerHub struct {
	server *httptest.Server

	mu sync.RWMutex

	repositories    map[string][]Repository // namespace -> repositories, insertion order
	tags            map[string][]Tag        // "namespace/repository" -> tags, insertion order
	pageLimit       int                     // when set, the most items a page carries
	requests        []Request               // every request received, in order
	statusOverrides map[string]int          // "METHOD /path" -> status to return
	bodyOverrides   map[string]string       // "METHOD /path" -> raw body to return
	failAfter       map[string]failAfter    // "METHOD /path" -> deferred failure
}

// Repository is a stored Docker Hub repository.
type Repository struct {
	Namespace string
	Name      string
	IsPrivate bool
}

// Tag is a stored tag of a repository.
type Tag struct {
	Name   string
	Status string // "active" or "inactive", as the API reports it
}

// NewDockerHub starts a fake Docker Hub API server and closes it on cleanup.
func NewDockerHub(t *testing.T) *DockerHub {
	t.Helper()

	fake := &DockerHub{
		repositories:    map[string][]Repository{},
		tags:            map[string][]Tag{},
		statusOverrides: map[string]int{},
		bodyOverrides:   map[string]string{},
		failAfter:       map[string]failAfter{},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v2/namespaces/{namespace}/repositories", fake.handleListRepositories)
	mux.HandleFunc("GET /v2/namespaces/{namespace}/repositories/{repository}", fake.handleGetRepository)
	mux.HandleFunc("GET /v2/namespaces/{namespace}/repositories/{repository}/tags", fake.handleListTags)
	mux.HandleFunc("GET /v2/namespaces/{namespace}/repositories/{repository}/tags/{tag}", fake.handleGetTag)

	fake.server = httptest.NewServer(fake.middleware(mux))
	t.Cleanup(fake.server.Close)

	return fake
}

// URL is the base URL of the fake, including the version path, suitable for a
// dockerhub Config BaseURL.
func (f *DockerHub) URL() string {
	return f.server.URL + "/v2"
}

// Close stops the fake before the end of the test, so that a caller sees a host
// that has gone away rather than one answering an error. The cleanup closes it
// again, which is harmless.
func (f *DockerHub) Close() {
	f.server.Close()
}

// --- Builder API ---

// AddRepository registers a public repository of a namespace.
func (f *DockerHub) AddRepository(namespace, name string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.repositories[namespace] = append(f.repositories[namespace], Repository{
		Namespace: namespace,
		Name:      name,
	})
}

// AddTag registers a tag of a repository. status may be empty, in which case
// the tag is active; the API reports inactive tags too, and callers are
// expected to leave them out.
func (f *DockerHub) AddTag(namespace, repository, tag, status string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if status == "" {
		status = "active"
	}

	key := namespace + "/" + repository
	f.tags[key] = append(f.tags[key], Tag{Name: tag, Status: status})
}

// --- Fault injection ---

// SetPageLimit makes every collection serve at most n items per page, however
// large a page the caller asks for, so that cursor-following is exercised.
func (f *DockerHub) SetPageLimit(n int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.pageLimit = n
}

// SetStatus makes every request to the given route answer with status. route is
// "METHOD /path", for example "GET /v2/namespaces/cimg/repositories".
func (f *DockerHub) SetStatus(route string, status int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.statusOverrides[route] = status
}

// SetBody makes every request to the given route answer with body verbatim.
func (f *DockerHub) SetBody(route, body string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.bodyOverrides[route] = body
}

// FailAfter makes a route succeed for its first n requests and then answer with
// status. Use it to fail a later page of a paginated read: a plain SetStatus
// would fail the first page too.
func (f *DockerHub) FailAfter(route string, n, status int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.failAfter[route] = failAfter{after: n, status: status}
}

// --- Inspection ---

// Requests returns every request the fake received, in order.
func (f *DockerHub) Requests() []Request {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return append([]Request(nil), f.requests...)
}

// RequestCount returns how many requests the fake received for a route.
func (f *DockerHub) RequestCount(method, path string) int {
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

func (f *DockerHub) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := map[string]string{}
		for key := range r.URL.Query() {
			query[key] = r.URL.Query().Get(key)
		}

		f.mu.Lock()
		f.requests = append(f.requests, Request{
			Method:    r.Method,
			Path:      r.URL.Path,
			Query:     query,
			UserAgent: r.Header.Get("User-Agent"),
		})
		route := r.Method + " " + r.URL.Path
		status := f.statusOverrides[route]
		body, hasBody := f.bodyOverrides[route]
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
			writeHubError(w, status)

			return
		}

		next.ServeHTTP(w, r)
	})
}

// --- Handlers ---

func (f *DockerHub) handleListRepositories(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")

	f.mu.RLock()
	repositories := f.repositories[namespace]
	f.mu.RUnlock()

	// A namespace nobody has heard of is an empty list, not a 404.
	results := make([]any, 0, len(repositories))
	for _, repository := range repositories {
		results = append(results, repositoryEntity(repository))
	}

	f.writePage(w, r, results)
}

func (f *DockerHub) handleGetRepository(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	name := r.PathValue("repository")

	f.mu.RLock()
	repositories := f.repositories[namespace]
	f.mu.RUnlock()

	for _, repository := range repositories {
		if repository.Name == name {
			writeJSON(w, http.StatusOK, repositoryEntity(repository))

			return
		}
	}

	writeHubError(w, http.StatusNotFound)
}

func (f *DockerHub) handleListTags(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	repository := r.PathValue("repository")

	// The tags of a repository Docker Hub does not have are a 404, not an
	// empty list: a caller cannot tell "no tags" from "no repository" unless
	// the fake keeps them apart.
	if !f.hasRepository(namespace, repository) {
		writeHubError(w, http.StatusNotFound)

		return
	}

	f.mu.RLock()
	tags := f.tags[namespace+"/"+repository]
	f.mu.RUnlock()

	// name narrows the list to tags containing it, which is what the real API
	// does and what tag completion relies on.
	name := r.URL.Query().Get("name")

	results := make([]any, 0, len(tags))
	for _, tag := range tags {
		if name != "" && !strings.Contains(tag.Name, name) {
			continue
		}
		results = append(results, map[string]any{
			"name":       tag.Name,
			"tag_status": tag.Status,
		})
	}

	f.writePage(w, r, results)
}

func (f *DockerHub) handleGetTag(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	repository := r.PathValue("repository")
	name := r.PathValue("tag")

	f.mu.RLock()
	tags := f.tags[namespace+"/"+repository]
	f.mu.RUnlock()

	for _, tag := range tags {
		if tag.Name == name {
			writeJSON(w, http.StatusOK, map[string]any{
				"name":       tag.Name,
				"tag_status": tag.Status,
			})

			return
		}
	}

	writeHubError(w, http.StatusNotFound)
}

// --- Rendering ---

// writePage serves results as a Docker Hub collection: a count, the results,
// and the absolute URL of the next page when there is one.
func (f *DockerHub) writePage(w http.ResponseWriter, r *http.Request, results []any) {
	page := 1
	if requested := r.URL.Query().Get("page"); requested != "" {
		parsed, err := strconv.Atoi(requested)
		if err != nil || parsed < 1 {
			writeHubError(w, http.StatusBadRequest)

			return
		}
		page = parsed
	}

	size := defaultPageSize
	if requested := r.URL.Query().Get("page_size"); requested != "" {
		parsed, err := strconv.Atoi(requested)
		if err != nil || parsed < 1 {
			writeHubError(w, http.StatusBadRequest)

			return
		}
		size = parsed
	}

	f.mu.RLock()
	limit := f.pageLimit
	f.mu.RUnlock()

	if limit > 0 && limit < size {
		size = limit
	}

	start := (page - 1) * size
	if start > len(results) {
		writeHubError(w, http.StatusNotFound)

		return
	}

	end := min(start+size, len(results))

	var next any
	if end < len(results) {
		next = f.pageURL(r, page+1, size)
	}

	var previous any
	if page > 1 {
		previous = f.pageURL(r, page-1, size)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"count":    len(results),
		"next":     next,
		"previous": previous,
		"results":  results[start:end],
	})
}

// pageURL is the absolute URL of a page of the collection being served,
// carrying over whatever else narrowed the request.
func (f *DockerHub) pageURL(r *http.Request, page, size int) string {
	query := r.URL.Query()
	query.Set("page", strconv.Itoa(page))
	query.Set("page_size", strconv.Itoa(size))

	return f.server.URL + r.URL.Path + "?" + query.Encode()
}

// hasRepository reports whether a namespace carries a repository.
func (f *DockerHub) hasRepository(namespace, name string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	for _, repository := range f.repositories[namespace] {
		if repository.Name == name {
			return true
		}
	}

	return false
}

func repositoryEntity(repository Repository) map[string]any {
	return map[string]any{
		"name":            repository.Name,
		"namespace":       repository.Namespace,
		"repository_type": "image",
		"status":          1,
		"is_private":      repository.IsPrivate,
	}
}

// writeHubError renders the {"message": ...} body Docker Hub reports errors as.
func writeHubError(w http.ResponseWriter, status int) {
	writeJSON(w, status, map[string]any{
		"message": "httperror " + strconv.Itoa(status) + ": " + http.StatusText(status),
	})
}
