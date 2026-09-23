package utils

import (
	"context"
	"fmt"
	"net/http"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
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
	var projectRes ProjectEnvVariableRes

	// The slug is joined onto the route as it is: its slashes are path
	// separators, which httpcl.RouteParams would escape.
	_, err := newV2Client(lsContext.Api).Call(context.Background(), httpcl.NewRequest(
		http.MethodGet, "/project/"+projectSlug+"/envvar",
		// The first page is asked for without a page-token at all.
		httpcl.OptionalQueryParam("page-token", nextPageToken),
		httpcl.JSONDecoder(&projectRes),
	))
	if err != nil {
		return nil, fmt.Errorf("list env vars of project %q: %w", projectSlug, err)
	}

	return &projectRes, nil
}
