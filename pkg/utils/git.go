package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-git/go-git/v5"
	gitUrl "github.com/whilp/git-urls"
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
		parsedUrl.Path = "/" + strings.TrimSuffix(parsedUrl.Path, ".git")
	}

	switch parsedUrl.Host {
	case "github.com":
		return "gh" + parsedUrl.Path
	case "bitbucket.org":
		return "bb" + parsedUrl.Path
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
	url := fmt.Sprintf("%s/api/v2/project/%s", lsContext.Api.HostUrl, projectSlug)

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("Circle-Token", lsContext.Api.Token)

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return Project{}, err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return Project{}, err
	}

	var projectIdRes Project
	err = json.Unmarshal(body, &projectIdRes)
	if err != nil {
		return Project{}, err
	}

	return projectIdRes, nil
}
