package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type ProjectEnvVariableRes struct {
	Items []struct {
		Name  string
		Value string
	}
	NextPageToken string `json:"next_page_token,omitempty"`
}

// GetAllProjectEnvVariables caches the names of a project's environment
// variables against the file that belongs to the project.
//
// Whatever was read before a failure is still cached, so a read that fails part
// way through the pages contributes the ones it got, and the error reports that
// the rest is missing. Nothing records whether the set is complete, unlike the
// context cache and its IsOrganizationContextListLoaded: a partial read looks
// like a project with fewer variables. That only costs completions that are
// absent rather than wrong, so the caller logs the error and carries on.
func GetAllProjectEnvVariables(lsContext *LsContext, cache *Cache, cachedFile *CachedFile) error {
	names, err := fetchAllProjectEnvVariables(lsContext, cachedFile.Project.Slug)

	for _, name := range names {
		cache.FileCache.AddEnvVariableToProjectLinkedToFile(cachedFile.TextDocument.URI, name)
	}

	return err
}

// fetchAllProjectEnvVariables reads the name of every environment variable of a
// project, following the page tokens. It reports the names it read before any
// failure, so a caller can use a partial answer.
func fetchAllProjectEnvVariables(lsContext *LsContext, projectSlug string) ([]string, error) {
	var names []string

	pageToken := ""

	for {
		res, err := getProjectEnvVariables(lsContext, projectSlug, pageToken)
		if err != nil {
			return names, err
		}

		for _, envVariable := range res.Items {
			names = append(names, envVariable.Name)
		}

		if res.NextPageToken == "" {
			return names, nil
		}

		pageToken = res.NextPageToken
	}
}

func getProjectEnvVariables(lsContext *LsContext, projectSlug string, nextPageToken string) (*ProjectEnvVariableRes, error) {
	requestUrl := fmt.Sprintf("%s/api/v2/project/%s/envvar", lsContext.Api.HostUrl, projectSlug)

	if nextPageToken != "" {
		// The route carries no other query parameter, so a later page is asked
		// for with "?" and not "&": with "&" the token became part of the path
		// and the request 404ed, which used to look like the end of the list.
		requestUrl += "?page-token=" + url.QueryEscape(nextPageToken)
	}

	req, err := http.NewRequest(http.MethodGet, requestUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Circle-Token", lsContext.Api.Token)
	req.Header.Set("User-Agent", UserAgent)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("list env vars of project %q: HTTP %d: %s", projectSlug, res.StatusCode, string(body))
	}

	var projectRes ProjectEnvVariableRes
	if err := json.Unmarshal(body, &projectRes); err != nil {
		return nil, err
	}

	return &projectRes, nil
}
