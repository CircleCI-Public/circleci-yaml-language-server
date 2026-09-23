package methods

import (
	"log/slog"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

func (methods *Methods) getAllEnvVariables(textDocument protocol.TextDocumentItem) {
	cachedFile := methods.Cache.FileCache.GetFile(textDocument.URI)
	if cachedFile.Project.Slug == "" {
		projectSlug := utils.GetProjectSlug(textDocument.URI.Filename())
		project, err := utils.GetProjectId(projectSlug, methods.LsContext)
		if err != nil {
			return
		}
		methods.Cache.FileCache.AddProjectSlugToFile(textDocument.URI, project)
		methods.updateProjectEnvVariables(cachedFile)
	}

	err := utils.GetAllContext(methods.LsContext, cachedFile.Project.OrganizationId, methods.Cache)
	if err != nil {
		slog.Warn("error getting contexts", "err", err)
		return
	}
	methods.Cache.ContextCache.MarkOrganizationContextListLoaded(cachedFile.Project.OrganizationId)

	if err := utils.GetAllContextWithEnvVars(methods.LsContext, cachedFile.Project.OrganizationId, methods.Cache); err != nil {
		slog.Warn("error getting context environment variables", "err", err)
	}
}

func (methods *Methods) updateProjectsEnvVariables() {
	for _, file := range methods.Cache.FileCache.GetFiles() {
		methods.updateProjectEnvVariables(file)
	}
}

func (methods *Methods) updateProjectEnvVariables(file *utils.CachedFile) {
	cachedFile := methods.Cache.FileCache.GetFile(file.TextDocument.URI)
	cachedFile.EnvVariables = []string{}
	methods.Cache.FileCache.SetFile(*cachedFile)
	// A file whose project is not resolved yet has no variables to read;
	// getAllEnvVariables reads them once it has resolved it.
	if cachedFile.Project.Slug == "" {
		return
	}
	if methods.LsContext.Api.Token != "" {
		if err := utils.GetAllProjectEnvVariables(methods.LsContext, methods.Cache, cachedFile); err != nil {
			slog.Warn("error getting project environment variables", "err", err)
		}
	}
}
