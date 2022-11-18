package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type Repository struct {
	Name           string `json:"name"`
	Namespace      string `json:"namespace"`
	RepositoryType string `json:"repository_type"`
	Status         int    `json:"status"`
	IsPrivate      bool   `json:"is_private"`
}

type HubResponse struct {
	Count    int          `json:"count"`
	Next     string       `json:"next"`
	Previous string       `json:"previous"`
	Results  []Repository `json:"results"`
}

// --
// Exposed
// --

type HubNamespace struct {
	namespace string
	nextURL   string
	hasLoaded bool // True if the namespace has been fetched at least once

	allRepositories []Repository
}

type SearchCusror struct {
	hub   *HubNamespace
	index int
	query string
}

type DockerResultsCursor interface {
	HasNext() bool
	Next() *Repository
	Prev() *Repository
}

func SearchDockerHUB(query string) DockerResultsCursor {
	// First, detect the namespace to search in
	namespace := getQueryNamespace(query)
	imageName := getQueryImageName(query)

	if hubNamespaces[namespace] == nil {
		hubNamespaces[namespace] = &HubNamespace{
			namespace: namespace,
		}
	}

	ns := hubNamespaces[namespace]

	return ns.createSearchCursor(imageName)
}

func DoesDockerImageExist(namespace, image, tag string) bool {
	// A quick win would be to check locally first, just in case we already found the image
	ns := hubNamespaces[namespace]

	if ns != nil && ns.hasLoaded {
		repo, _ := findFirstByName(&ns.allRepositories, image)

		if repo != nil {
			return true
		}
	}

	url := baseURL.JoinPath(
		fmt.Sprintf("namespaces/%s/repositories/%s", namespace, image),
	)

	if tag != "" && tag != "latest" {
		// "Latest" is a keyword and is not fetchable via API.
		// in that case, the image is valid as long as a repo with the name exists.

		url = url.JoinPath(
			fmt.Sprintf("/tags/%s", tag),
		)
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return false
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}

	return res.StatusCode == 200
}

// --
// Private
// --

var baseURL = url.URL{
	Scheme: "https",
	Host:   "hub.docker.com",
	Path:   "v2",
}

var hubNamespaces = map[string]*HubNamespace{
	"library": {
		namespace: "library",
	},
}

var repositoriesTags = map[string]*string{}

var namespaceRegex = regexp.MustCompile(`^([a-z0-9\-_]+)\/`)
var tagNameRegex = regexp.MustCompile(`(?:[a-z0-9\-_]+\/)?(?:[a-z0-9\-_]+):(.*)`)
var imageNameRegex = regexp.MustCompile(`^([a-z0-9\-_]+\/([a-z0-9\-_]+)|[a-z0-9\-_]+).*$`)

func (h *HubNamespace) createSearchCursor(search string) DockerResultsCursor {
	return &SearchCusror{
		hub:   h,
		index: -1,
		query: search,
	}
}

func (h *HubNamespace) loadNext() ([]Repository, error) {
	hubResponse := HubResponse{}
	queryURL := h.nextURL

	if !h.hasLoaded {
		queryURL = baseURL.JoinPath(
			fmt.Sprintf("/namespaces/%s/repositories", h.namespace),
		).String()
	} else if h.hasLoaded && h.nextURL == "" {
		return nil, fmt.Errorf("No more to load")
	}

	fmt.Println(
		fmt.Sprintf("DOCKERHUB: Namespace fetch %s (%s)", h.namespace, queryURL),
	)

	req, err := http.NewRequest("GET", queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to load next")
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to load next")
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to load next")
	}

	err = json.Unmarshal(body, &hubResponse)
	if err != nil {
		return nil, err
	}

	h.allRepositories = append(h.allRepositories, hubResponse.Results...)
	h.nextURL = hubResponse.Next
	h.hasLoaded = true

	return hubResponse.Results, nil
}

// --
// Implement Cursor for search results
// --

func (s *SearchCusror) HasNext() bool {
	start := s.index + 1
	searchItems := s.hub.allRepositories[start:]
	_, index := findFirstMatch(&searchItems, s.query)

	if index >= 0 {
		return true
	}

	// Otherwhise, let the result check if there can potentially be a next result & then find ...
	for s.hub.nextURL != "" || !s.hub.hasLoaded {
		s.hub.loadNext()
		searchItems := s.hub.allRepositories[start:]

		_, index = findFirstMatch(
			&searchItems,
			s.query,
		)
	}

	return index >= 0
}

func (s *SearchCusror) Next() *Repository {
	start := s.index + 1
	searchDomain := s.hub.allRepositories[start:] // TODO: Check Bounds
	repo, index := findFirstMatch(&searchDomain, s.query)

	if index >= 0 {
		s.index += index + 1
	}

	return repo
}

func (s *SearchCusror) Prev() *Repository {
	searchDomain := s.hub.allRepositories[:s.index] // TODO: Check bounds
	repo, index := findFirstMatch(&searchDomain, s.query)

	if index >= 0 {
		s.index -= len(searchDomain) - index
	}

	return repo
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

func getQueryTagName(query string) string {
	matches := tagNameRegex.FindAllStringSubmatch(query, -1)

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
