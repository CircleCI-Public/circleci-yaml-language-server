package dockerhub

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type TagResponse struct {
	BaseHUBResponse
	Results []RepoTag `json:"results"`
}

type RepoTag struct {
	Name string `json:"name"`
}

func (t *TagResponse) loadNext() (TagResponse, error) {
	if t.Next == "" {
		return TagResponse{}, fmt.Errorf("Failed to fetch more tags: nothing to fetch")
	}

	return fetchTagsByURL(t.Next)
}

func fetchTags(namespace, repo, name string) (TagResponse, error) {
	url := baseURL.JoinPath(
		fmt.Sprintf("/namespaces/%s/repositories/%s/tags", namespace, repo),
	)

	q := url.Query()
	q.Add("page_size", "100")

	if name != "" {
		q.Add("name", name)
	}

	url.RawQuery = q.Encode()

	queryURL := url.String()

	return fetchTagsByURL(queryURL)
}

func fetchTagsByURL(queryURL string) (TagResponse, error) {
	tagResponse := TagResponse{}
	req, err := http.NewRequest("GET", queryURL, nil)
	if err != nil {
		return tagResponse, fmt.Errorf("Failed to load next")
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return tagResponse, fmt.Errorf("Failed to load next")
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return tagResponse, fmt.Errorf("Failed to load next")
	}

	err = json.Unmarshal(body, &tagResponse)
	if err != nil {
		return tagResponse, err
	}

	return tagResponse, nil
}
