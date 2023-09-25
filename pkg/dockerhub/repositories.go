package dockerhub

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
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
	namespace string
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
var tagNameRegex = regexp.MustCompile(`(?:[a-z0-9\-_]+\/)?(?:[a-z0-9\-_]+):(.*)`)
var imageNameRegex = regexp.MustCompile(`^([a-z0-9\-_]+\/([a-z0-9\-_]+)|[a-z0-9\-_]+).*$`)

func (h *HubNamespace) createSearchCursor(search string) DockerResultsCursor {
	return &SearchCursor{
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

	req, err := http.NewRequest("GET", queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to load next")
	}

	req.Header.Set("User-Agent", utils.UserAgent)

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
