package dockerhub

import (
	"fmt"
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

func (t *TagResponse) loadNext(api *dockerHubAPI) (TagResponse, error) {
	if t.Next == "" {
		return TagResponse{}, fmt.Errorf("Failed to fetch more tags: nothing to fetch")
	}

	return api.fetchTagsByURL(t.Next)
}

func (me *dockerHubAPI) fetchTags(namespace, repo, name string) (TagResponse, error) {
	url := me.baseURL.JoinPath(
		fmt.Sprintf("/namespaces/%s/repositories/%s/tags", namespace, repo),
	)

	q := url.Query()
	q.Add("page_size", "100")

	if name != "" {
		q.Add("name", name)
	}

	url.RawQuery = q.Encode()

	queryURL := url.String()

	return me.fetchTagsByURL(queryURL)
}

func (me *dockerHubAPI) fetchTagsByURL(queryURL string) (TagResponse, error) {
	tagResponse := TagResponse{}
	if _, err := me.get(queryURL, &tagResponse); err != nil {
		return TagResponse{}, fmt.Errorf("fetching tags: %w", err)
	}

	return tagResponse, nil
}

func (me *dockerHubAPI) GetImageTags(namespace, image string) ([]string, error) {
	url := me.baseURL.JoinPath(
		fmt.Sprintf("namespaces/%s/repositories/%s/tags", namespace, image),
	)

	body := TagResponse{}
	if _, err := me.get(url.String(), &body); err != nil {
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

	status, err := me.get(url.String(), nil)

	return err == nil && status == http.StatusOK
}
