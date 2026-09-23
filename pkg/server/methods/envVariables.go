package methods

import (
	"log/slog"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/projectslug"
	"go.lsp.dev/protocol"
)

func (methods *Methods) getAllEnvVariables(textDocument protocol.TextDocumentItem) {
	cachedFile := methods.Cache.FileCache.GetFile(textDocument.URI)
	if cachedFile.Project.Slug == "" {
		projectSlug := projectslug.FromRepo(textDocument.URI.Filename())
		project, err := circleci.GetProject(methods.Settings.Api, projectSlug)
		if err != nil {
			return
		}
		methods.Cache.FileCache.AddProjectSlugToFile(textDocument.URI, project)
		methods.updateProjectEnvVariables(cachedFile)
	}

	err := methods.Cache.LoadContexts(methods.Settings.Api, cachedFile.Project.OrganizationId)
	if err != nil {
		slog.Warn("error getting contexts", "err", err)
		return
	}
	methods.Cache.ContextCache.MarkOrganizationContextListLoaded(cachedFile.Project.OrganizationId)

	if err := methods.Cache.LoadContextEnvVariables(methods.Settings.Api, cachedFile.Project.OrganizationId); err != nil {
		slog.Warn("error getting context environment variables", "err", err)
	}
}

func (methods *Methods) updateProjectsEnvVariables() {
	for _, file := range methods.Cache.FileCache.GetFiles() {
		methods.updateProjectEnvVariables(file)
	}
}

func (methods *Methods) updateProjectEnvVariables(file *cache.File) {
	cachedFile := methods.Cache.FileCache.GetFile(file.TextDocument.URI)
	cachedFile.EnvVariables = []string{}
	methods.Cache.FileCache.SetFile(*cachedFile)
	// A file whose project is not resolved yet has no variables to read;
	// getAllEnvVariables reads them once it has resolved it.
	if cachedFile.Project.Slug == "" {
		return
	}
	if methods.Settings.Api.Token != "" {
		if err := methods.Cache.LoadProjectEnvVariables(methods.Settings.Api, cachedFile); err != nil {
			slog.Warn("error getting project environment variables", "err", err)
		}
	}
}
