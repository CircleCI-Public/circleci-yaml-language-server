package utils

import (
	"net/url"
	"strings"

	"github.com/go-git/go-git/v5"
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
		return ""
	}

	switch parsedUrl.Host {
	case "github.com":
		return "gh" + parsedUrl.Path
	case "bitbucket.org":
		return "bb" + parsedUrl.Path
	}

	return ""
}
