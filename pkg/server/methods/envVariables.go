package methods

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func (methods *Methods) getAllEnvVariables(textDocument protocol.TextDocumentItem) {
	cachedFile := methods.Cache.FileCache.GetFile(textDocument.URI)
	if cachedFile.ProjectSlug == "" {
		projectSlug := utils.GetProjectSlug(textDocument.URI.Filename())
		methods.Cache.FileCache.AddProjectSlugToFile(textDocument.URI, projectSlug)
		methods.updateProjectEnvVariables(cachedFile)
	}

	hasBeenUpdated, _ := utils.GetAllContext(methods.LsContext, methods.LsContext.Api.GetUserId(), "", methods.Cache)
	if hasBeenUpdated {
		utils.GetAllContextAllEnvVariables(methods.LsContext, methods.Cache)
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
	if methods.LsContext.Api.Token != "" {
		utils.GetAllProjectEnvVariables(methods.LsContext, methods.Cache, cachedFile)
	}
}
