package projectslug

import (
	"net/url"
	"strings"

	gitUrl "github.com/chainguard-dev/git-urls"
	"github.com/go-git/go-git/v6"
)

// RepoDir is the directory of the repository a config belongs to: the one
// holding its .circleci directory.
func RepoDir(configPath string) string {
	return strings.Split(configPath, ".circleci")[0]
}

// FromRepo returns the project slug the remote of a config's repository names,
// or "" when it names none: origin, or the only remote there is.
func FromRepo(configPath string) string {
	repo, err := git.PlainOpen(RepoDir(configPath))
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

// OrgSlug returns the slug of a project's organization, such as "gh/acme" for
// "gh/acme/rocket", or "" when projectSlug is not a project slug.
func OrgSlug(projectSlug string) string {
	splitted := strings.Split(projectSlug, "/")

	if len(splitted) != 3 {
		return ""
	}

	return splitted[0] + "/" + splitted[1]
}
