package cache

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
)

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
