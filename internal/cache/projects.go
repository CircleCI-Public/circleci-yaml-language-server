package cache

import (
	"net/http"
	"time"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/memo"
)

// Projects remembers the project each slug names, and that a slug names none.
// Every file of a repository names the same project, and each is resolved as
// it is opened.
type Projects struct {
	// projects holds the zero Project for a slug that names none.
	projects *memo.Memo[circleci.Project]
}

func projectLifetime(project circleci.Project) time.Duration {
	return memo.Existence(project.Slug != "")
}

// Project returns the project a slug names, or the zero Project when it names
// none, asking the host only when no answer is remembered. An error is
// returned but not remembered.
//
// A repository that is not a CircleCI project is common, and is asked about
// on every edit of its config, so that it names no project is remembered too.
func (c *Cache) Project(api circleci.Config, slug string) (circleci.Project, error) {
	return c.ProjectCache.projects.Get(slug, func() (circleci.Project, error) {
		project, err := circleci.GetProject(api, slug)
		if httpcl.HasStatusCode(err, http.StatusNotFound) {
			return circleci.Project{}, nil
		}
		return project, err
	})
}

// LoadProjectEnvVariables caches the names of a project's environment
// variables against the file that belongs to the project.
//
// Whatever was read before a failure is still cached, so a read that fails part
// way through the pages contributes the ones it got, and the error reports that
// the rest is missing. Nothing records whether the set is complete, unlike the
// context cache and its IsOrganizationContextListLoaded: a partial read looks
// like a project with fewer variables. That only costs completions that are
// absent rather than wrong, so the caller logs the error and carries on.
func (c *Cache) LoadProjectEnvVariables(api circleci.Config, cachedFile *File) error {
	names, err := circleci.ListProjectEnvVarNames(api, cachedFile.Project.Slug)

	for _, name := range names {
		c.FileCache.AddEnvVariableToProjectLinkedToFile(cachedFile.TextDocument.URI, name)
	}

	return err
}
