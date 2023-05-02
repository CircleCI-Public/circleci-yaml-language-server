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
	TagStatus string `json:"tag_status"`
	Name      string `json:"name"`
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

func (me *dockerHubAPI) GetImageTags(namespace, image string) ([]string, error) {
	url := me.baseURL.JoinPath(
		fmt.Sprintf("namespaces/%s/repositories/%s/tags", namespace, image),
	)

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(res.Body)
	body := TagResponse{}

	err = decoder.Decode(&body)
	if err != nil {
		return nil, err
	}

	tags := make([]string, len(body.Results))
	for i, tag := range body.Results {
		// Although there is no documentation about this field in the doc, all tags I came across were
		// tagged as "active" so it feels like it should be verified
		// https://docs.docker.com/docker-hub/api/latest/#tag/repositories/paths/~1v2~1namespaces~1%7Bnamespace%7D~1repositories~1%7Brepository%7D~1tags/get
		if tag.TagStatus == "active" {
			tags[i] = tag.Name
		}
	}

	return tags, nil
}

func (me *dockerHubAPI) ImageHasTag(namespace, image, tag string) bool {
	url := me.baseURL.JoinPath(
		fmt.Sprintf("namespaces/%s/repositories/%s/tags/%s", namespace, image, tag),
	)

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
