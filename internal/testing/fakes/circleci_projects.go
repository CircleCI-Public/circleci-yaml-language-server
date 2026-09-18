package fakes

// This file serves projects and their environment variables: the routes the
// language server reads a config's project and organization from, and the
// project variables it offers in completion.

import (
	"net/http"
	"strings"
)

// projectsState is the stored projects and their environment variables.
type projectsState struct {
	projects map[string]Project         // project slug -> project
	envVars  map[string][]ProjectEnvVar // project slug -> env vars, insertion order
}

func newProjectsState() projectsState {
	return projectsState{
		projects: map[string]Project{},
		envVars:  map[string][]ProjectEnvVar{},
	}
}

// Project is a stored project.
type Project struct {
	Slug     string
	ID       string
	OrgID    string
	OrgSlug  string
	Name     string
	OrgName  string
	Provider string
	VcsURL   string
}

// ProjectEnvVar is a stored environment variable of a project.
type ProjectEnvVar struct {
	Name  string
	Value string
}

// AddProject registers a project. The project name, organization name and VCS
// information are derived from the slug and the organization slug the way the
// real API reports them, so that a caller only has to state the identifiers it
// cares about.
func (f *CircleCI) AddProject(slug, id, orgID, orgSlug string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	project := Project{
		Slug:    slug,
		ID:      id,
		OrgID:   orgID,
		OrgSlug: orgSlug,
		Name:    lastSegment(slug),
		OrgName: lastSegment(orgSlug),
	}

	switch {
	case strings.HasPrefix(slug, "gh/"):
		project.Provider = "GitHub"
		project.VcsURL = "https://github.com/" + strings.TrimPrefix(slug, "gh/")
	case strings.HasPrefix(slug, "bb/"):
		project.Provider = "Bitbucket"
		project.VcsURL = "https://bitbucket.org/" + strings.TrimPrefix(slug, "bb/")
	}

	f.projects.projects[slug] = project
}

// AddProjectEnvVar registers an environment variable of a project. value is
// what the API reports in place of the real one, which it truncates; pass ""
// for a generated one.
func (f *CircleCI) AddProjectEnvVar(slug, name, value string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if value == "" {
		value = "xxxx1234"
	}

	f.projects.envVars[slug] = append(f.projects.envVars[slug], ProjectEnvVar{
		Name:  name,
		Value: value,
	})
}

func (f *CircleCI) handleGetProject(w http.ResponseWriter, r *http.Request) {
	slug := projectSlug(r)

	f.mu.RLock()
	project, ok := f.projects.projects[slug]
	f.mu.RUnlock()

	if !ok {
		writeV2Error(w, http.StatusNotFound, "Project not found")

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"slug":              project.Slug,
		"name":              project.Name,
		"id":                project.ID,
		"organization_name": project.OrgName,
		"organization_slug": project.OrgSlug,
		"organization_id":   project.OrgID,
		"vcs_info": map[string]any{
			"vcs_url":        project.VcsURL,
			"provider":       project.Provider,
			"default_branch": "main",
		},
	})
}

func (f *CircleCI) handleGetProjectEnvVars(w http.ResponseWriter, r *http.Request) {
	slug := projectSlug(r)

	f.mu.RLock()
	_, known := f.projects.projects[slug]
	envVars := f.projects.envVars[slug]
	f.mu.RUnlock()

	if !known {
		writeV2Error(w, http.StatusNotFound, "Project not found")

		return
	}

	items := make([]any, 0, len(envVars))
	for _, envVar := range envVars {
		items = append(items, map[string]any{
			"name":  envVar.Name,
			"value": envVar.Value,
		})
	}

	f.writeV2Page(w, r, "project/envvar", items)
}

// projectSlug reassembles the three path segments of a project slug.
func projectSlug(r *http.Request) string {
	return r.PathValue("vcs") + "/" + r.PathValue("org") + "/" + r.PathValue("project")
}

func lastSegment(path string) string {
	segments := strings.Split(path, "/")

	return segments[len(segments)-1]
}
