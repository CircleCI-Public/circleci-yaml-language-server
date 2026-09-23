package dockerhub

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

type BaseHUBResponse struct {
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
}

type Repository struct {
	Name           string `json:"name"`
	Namespace      string `json:"namespace"`
	RepositoryType string `json:"repository_type"`
	Status         int    `json:"status"`
	IsPrivate      bool   `json:"is_private"`

	Tags []RepoTag
}

type HubResponse struct {
	BaseHUBResponse
	Results []Repository `json:"results"`
}

// --
// Exposed
// --

type HubNamespace struct {
	api       *dockerHubAPI
	namespace string

	// mutex guards everything below. Completion searches a namespace from
	// several goroutines at once, so a search holds it for as long as it
	// walks, loads included: a second search waits for the first to finish
	// reading a page rather than asking Docker Hub for the same one.
	mutex     sync.Mutex
	nextURL   string
	hasLoaded bool // True if the namespace has been fetched at least once

	allRepositories []Repository
}

// --
// Private
// --

var baseURL = url.URL{
	Scheme: "https",
	Host:   "hub.docker.com",
	Path:   "v2",
}

var namespaceRegex = regexp.MustCompile(`^([a-z0-9\-_]+)\/`)
var imageNameRegex = regexp.MustCompile(`^([a-z0-9\-_]+\/([a-z0-9\-_]+)|[a-z0-9\-_]+).*$`)

func (h *HubNamespace) createSearchCursor(search string) ResultsCursor {
	return &SearchCursor{
		hub:   h,
		index: -1,
		query: search,
	}
}

// hasRepository reports whether a repository of that name has already been
// read, without asking Docker Hub.
func (h *HubNamespace) hasRepository(name string) bool {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if !h.hasLoaded {
		return false
	}

	repo, _ := findFirstByName(&h.allRepositories, name)

	return repo != nil
}

// loadNext reads the next page of the namespace. The caller holds h.mutex.
func (h *HubNamespace) loadNext() ([]Repository, error) {
	hubResponse := HubResponse{}
	queryURL := h.nextURL

	if !h.hasLoaded {
		queryURL = h.api.baseURL.JoinPath(
			fmt.Sprintf("/namespaces/%s/repositories", h.namespace),
		).String()
	} else if h.hasLoaded && h.nextURL == "" {
		return nil, fmt.Errorf("no more to load")
	}

	if _, err := h.api.get(queryURL, &hubResponse); err != nil {
		return nil, fmt.Errorf("loading repositories of %s: %w", h.namespace, err)
	}

	h.allRepositories = append(h.allRepositories, hubResponse.Results...)
	h.nextURL = hubResponse.Next
	h.hasLoaded = true

	return hubResponse.Results, nil
}

func getQueryNamespace(query string) string {
	matches := namespaceRegex.FindAllStringSubmatch(query, -1)

	if len(matches) == 0 {
		return "library"
	}

	return matches[0][1]
}

func getQueryImageName(query string) string {
	if namespaceRegex.MatchString(query) {
		matches := imageNameRegex.FindAllStringSubmatch(query, -1)

		if len(matches) == 0 {
			return ""
		}

		return matches[0][2]
	}

	matches := imageNameRegex.FindAllStringSubmatch(query, -1)

	if len(matches) == 0 {
		return ""
	}

	return matches[0][1]
}

func findFirstMatch(repositories *[]Repository, name string) (*Repository, int) {
	for index, repo := range *repositories {
		if strings.HasPrefix(repo.Name, name) {
			return &repo, index
		}
	}

	return nil, -1
}

func findFirstByName(repositories *[]Repository, name string) (*Repository, int) {
	for index, repo := range *repositories {
		if repo.Name == name {
			return &repo, index
		}
	}

	return nil, -1
}
