package methods

import (
	"log/slog"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/projectslug"
)

func (methods *Methods) getAllEnvVariables(textDocument protocol.TextDocumentItem) {
	api := methods.Settings().Api
	cachedFile := methods.Cache.FileCache.GetFile(textDocument.URI)
	if cachedFile == nil {
		return
	}
	if cachedFile.Project.Slug == "" {
		projectSlug := projectslug.FromRepo(textDocument.URI.FsPath())
		project, err := circleci.GetProject(api, projectSlug)
		if err != nil {
			return
		}
		methods.Cache.FileCache.AddProjectSlugToFile(textDocument.URI, project)
		cachedFile.Project = project
		methods.updateProjectEnvVariables(cachedFile)
	}

	err := methods.Cache.LoadContexts(api, cachedFile.Project.OrganizationId)
	if err != nil {
		slog.Warn("error getting contexts", "err", err)
		return
	}
	methods.Cache.ContextCache.MarkOrganizationContextListLoaded(cachedFile.Project.OrganizationId)

	if err := methods.Cache.LoadContextEnvVariables(api, cachedFile.Project.OrganizationId); err != nil {
		slog.Warn("error getting context environment variables", "err", err)
	}
}

func (methods *Methods) updateProjectsEnvVariables() {
	for _, file := range methods.Cache.FileCache.GetFiles() {
		methods.updateProjectEnvVariables(file)
	}
}

func (methods *Methods) updateProjectEnvVariables(file *cache.File) {
	methods.Cache.FileCache.ClearEnvVariables(file.TextDocument.URI)
	cachedFile := methods.Cache.FileCache.GetFile(file.TextDocument.URI)
	if cachedFile == nil {
		return
	}
	// A file whose project is not resolved yet has no variables to read;
	// getAllEnvVariables reads them once it has resolved it.
	if cachedFile.Project.Slug == "" {
		return
	}
	if api := methods.Settings().Api; api.Token != "" {
		if err := methods.Cache.LoadProjectEnvVariables(api, cachedFile); err != nil {
			slog.Warn("error getting project environment variables", "err", err)
		}
	}
}
