package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ProjectEnvVariableRes struct {
	Items []struct {
		Name  string
		Value string
	}
	NextPageToken string `json:"next_page_token,omitempty"`
}

func GetAllProjectEnvVariables(lsContext *LsContext, cache *Cache, cachedFile *CachedFile) {
	var projectEnvVariables []string

	fetchAllProjectEnvVariables(lsContext, cachedFile.Project.Slug, "", cache, &projectEnvVariables)

	for _, projectEnvVariable := range projectEnvVariables {
		cache.FileCache.AddEnvVariableToProjectLinkedToFile(cachedFile.TextDocument.URI, projectEnvVariable)
	}
}

func fetchAllProjectEnvVariables(lsContext *LsContext, projectSlug string, nextPageToken string, cache *Cache, projectEnvVariables *[]string) error {
	var nextPageQuery string

	if nextPageToken != "" {
		nextPageQuery = fmt.Sprintf("&page-token=%s", nextPageToken)
	} else {
		nextPageQuery = ""
	}
	url := fmt.Sprintf("%s/api/v2/project/%s/envvar%s", lsContext.Api.HostUrl, projectSlug, nextPageQuery)
	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("Circle-Token", lsContext.Api.Token)
	req.Header.Set("User-Agent", UserAgent)

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var projectRes ProjectEnvVariableRes
	err = json.Unmarshal(body, &projectRes)
	if err != nil {
		return err
	}

	for _, project := range projectRes.Items {
		*projectEnvVariables = append(*projectEnvVariables, project.Name)
	}

	if projectRes.NextPageToken != "" {
		return fetchAllProjectEnvVariables(lsContext, projectSlug, projectRes.NextPageToken, cache, projectEnvVariables)
	}

	return nil
}
