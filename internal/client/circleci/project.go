package circleci

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

// ListProjectEnvVarNames reads the name of every environment variable of a
// project, following the page tokens. It reports the names it read before any
// failure, so a caller can use a partial answer.
func ListProjectEnvVarNames(api Config, projectSlug string) ([]string, error) {
	var names []string

	pageToken := ""

	for {
		res, err := getProjectEnvVariables(api, projectSlug, pageToken)
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

func getProjectEnvVariables(api Config, projectSlug string, nextPageToken string) (*ProjectEnvVariableRes, error) {
	var projectRes ProjectEnvVariableRes

	// The slug is joined onto the route as it is: its slashes are path
	// separators, which httpcl.RouteParams would escape.
	_, err := newV2Client(api).Call(context.Background(), httpcl.NewRequest(
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

type Project struct {
	Slug             string
	Name             string
	Id               string
	OrganizationName string `json:"organization_name"`
	OrganizationSlug string `json:"organization_slug"`
	OrganizationId   string `json:"organization_id"`
	VcsInfo          struct {
		VcsUrl         string `json:"vcs_url"`
		Provider       string
		Default_branch string `json:"default_branch"`
	} `json:"vcs_info"`
}

// GetProject reads the project a slug names, which is how its organization id
// is found.
func GetProject(api Config, projectSlug string) (Project, error) {
	var projectIdRes Project

	// The slug is joined onto the route as it is: its slashes are path
	// separators, which httpcl.RouteParams would escape.
	_, err := newV2Client(api).Call(context.Background(), httpcl.NewRequest(
		http.MethodGet, "/project/"+projectSlug,
		httpcl.JSONDecoder(&projectIdRes),
	))
	if err != nil {
		return Project{}, fmt.Errorf("get project %q: %w", projectSlug, err)
	}

	return projectIdRes, nil
}
