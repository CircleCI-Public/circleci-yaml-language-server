package utils

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-git/go-git/v5"
	gitUrl "github.com/whilp/git-urls"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
)

func GetProjectSlug(configPath string) string {
	repo, err := git.PlainOpen(strings.Split(configPath, ".circleci")[0])
	if err != nil {
		return ""
	}

	remotes, err := repo.Remotes()
	if err != nil || len(remotes) == 0 {
		return ""
	}

	if len(remotes) == 1 {
		return fromUrlToProjectSlug(remotes[0].Config().URLs[0])
	}

	for _, remote := range remotes {
		if remote.Config().Name == "origin" {
			return fromUrlToProjectSlug(remote.Config().URLs[0])
		}
	}

	return ""
}

func fromUrlToProjectSlug(projectUrl string) string {
	parsedUrl, err := url.Parse(projectUrl)
	if err != nil {
		parsedUrl, err = gitUrl.ParseScp(projectUrl)
		if err != nil {
			return ""
		}
	}

	// Every form of remote can end in .git — it is what `git clone` leaves in
	// origin, over https as much as over ssh — and no project slug does. The
	// scp form's path has no leading slash, and the others' do.
	path := "/" + strings.TrimPrefix(strings.TrimSuffix(parsedUrl.Path, ".git"), "/")

	switch parsedUrl.Hostname() {
	case "github.com":
		return "gh" + path
	case "bitbucket.org":
		return "bb" + path
	}

	return ""
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

func GetProjectOrg(projectSlug string) string {
	splitted := strings.Split(projectSlug, "/")

	if len(splitted) != 3 {
		return ""
	}

	return splitted[1]
}

func GetProjectId(projectSlug string, lsContext *LsContext) (Project, error) {
	var projectIdRes Project

	// The slug is joined onto the route as it is: its slashes are path
	// separators, which httpcl.RouteParams would escape.
	_, err := newV2Client(lsContext.Api).Call(context.Background(), httpcl.NewRequest(
		http.MethodGet, "/project/"+projectSlug,
		httpcl.JSONDecoder(&projectIdRes),
	))
	if err != nil {
		return Project{}, fmt.Errorf("get project %q: %w", projectSlug, err)
	}

	return projectIdRes, nil
}
