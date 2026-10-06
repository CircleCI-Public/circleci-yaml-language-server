package methods

import (
	"context"
	"log/slog"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
)

func (methods *Methods) getAllEnvVariables(ctx context.Context, textDocument protocol.TextDocumentItem) {
	client := methods.Settings().Client()
	cachedFile := methods.Cache.FileCache.GetFile(textDocument.URI)
	if cachedFile == nil {
		return
	}
	if cachedFile.Project.Slug == "" {
		projectSlug := methods.Cache.ProjectSlugOfFile(ctx, textDocument.URI.FsPath())
		if projectSlug == "" {
			return
		}
		project, err := methods.Cache.Project(ctx, client, projectSlug)
		if err != nil || project.Slug == "" {
			return
		}
		methods.Cache.FileCache.AddProjectSlugToFile(textDocument.URI, project)
		cachedFile.Project = project
		methods.updateProjectEnvVariables(ctx, cachedFile)
	}

	if err := methods.Cache.LoadContexts(ctx, client, cachedFile.Project.OrganizationId); err != nil {
		slog.Warn("error getting contexts", "err", err)
	}
}

func (methods *Methods) updateProjectsEnvVariables(ctx context.Context) {
	for _, file := range methods.Cache.FileCache.GetFiles() {
		methods.updateProjectEnvVariables(ctx, file)
	}
}

func (methods *Methods) updateProjectEnvVariables(ctx context.Context, file *cache.File) {
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
	if settings := methods.Settings(); settings.Api.Token != "" {
		if err := methods.Cache.LoadProjectEnvVariables(ctx, settings.Client(), cachedFile); err != nil {
			slog.Warn("error getting project environment variables", "err", err)
		}
	}
}
