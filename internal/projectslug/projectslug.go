package projectslug

import (
	"net/url"
	"strings"

	gitUrl "github.com/chainguard-dev/git-urls"
	"github.com/go-git/go-git/v6"
)

func FromRepo(configPath string) string {
	repo, err := git.PlainOpen(strings.Split(configPath, ".circleci")[0])
	if err != nil {
		return ""
	}

	remotes, err := repo.Remotes()
	if err != nil || len(remotes) == 0 {
		return ""
	}

	if len(remotes) == 1 {
		return fromURL(remotes[0].Config().URLs[0])
	}

	for _, remote := range remotes {
		if remote.Config().Name == "origin" {
			return fromURL(remote.Config().URLs[0])
		}
	}

	return ""
}

func fromURL(projectUrl string) string {
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

func Org(projectSlug string) string {
	splitted := strings.Split(projectSlug, "/")

	if len(splitted) != 3 {
		return ""
	}

	return splitted[1]
}
