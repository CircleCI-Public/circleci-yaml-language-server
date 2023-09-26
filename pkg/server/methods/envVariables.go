package methods

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
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

	utils.GetAllContext(methods.LsContext, cachedFile.Project.OrganizationName, cachedFile.Project.VcsInfo.Provider, methods.Cache)
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
