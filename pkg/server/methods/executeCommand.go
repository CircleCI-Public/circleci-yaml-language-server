package methods

import (
	"fmt"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"github.com/segmentio/encoding/json"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

func (methods *Methods) ExecuteCommand(reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	params := protocol.ExecuteCommandParams{}
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(methods.Ctx, nil, fmt.Errorf("%s: %w", jsonrpc2.ErrParse, err))
	}

	arguments := params.Arguments

	switch params.Command {
	case "setToken":
		param, ok := arguments[0].(string)
		if !ok {
			return reply(methods.Ctx, nil, nil)
		}
		methods.setToken(param)

	case "setSelfHostedUrl":
		param, ok := arguments[0].(string)
		if !ok {
			return reply(methods.Ctx, nil, nil)
		}
		methods.setHostUrl(param)
	case "setUserId":
		param, ok := arguments[0].(string)
		if !ok {
			return reply(methods.Ctx, nil, nil)
		}
		methods.setUserId(param)

	case "setProjectSlug":
		param, ok := arguments[0].(string)
		if !ok {
			return reply(methods.Ctx, nil, nil)
		}
		methods.setProjectSlug(param)

	}

	return reply(methods.Ctx, nil, nil)
}

func (methods *Methods) setToken(token string) {
	if methods.LsContext.Api.Token != token {
		methods.Cache.ClearHostData()
	}

	methods.LsContext.Api.Token = token
	filesCache := methods.Cache.FileCache.GetFiles()
	for _, file := range filesCache {
		go methods.notificationMethods(methods.Cache.FileCache, *file)
	}

	methods.updateProjectsEnvVariables()
}

func (methods *Methods) setHostUrl(hostUrl string) {
	if methods.LsContext.Api.HostUrl != hostUrl {
		methods.Cache.ClearHostData()
	}

	if hostUrl != "" {
		methods.LsContext.Api.HostUrl = hostUrl
	} else {
		methods.LsContext.Api.HostUrl = utils.CIRCLE_CI_APP_HOST_URL
	}

	filesCache := methods.Cache.FileCache.GetFiles()
	for _, file := range filesCache {
		go methods.notificationMethods(methods.Cache.FileCache, *file)
	}

	methods.updateProjectsEnvVariables()
}

func (methods *Methods) setUserId(userId string) {
	methods.LsContext.UserId = userId
}

func (methods *Methods) setProjectSlug(projectSlug string) {
	if projectSlug == "" {
		return
	}

	methods.Cache.ProjectCache.SetProject(&utils.Project{
		Slug: projectSlug,
	})
	if methods.LsContext.Api.Token != "" {
		utils.GetAllProjectEnvVariables(methods.LsContext, methods.Cache, projectSlug)
	}
}

func (methods *Methods) updateProjectsEnvVariables() {
	for _, project := range methods.Cache.ProjectCache.GetAllProjects() {
		methods.updateProjectEnvVariables(project.Slug)
	}
}

func (methods *Methods) updateProjectEnvVariables(projectSlug string) {
	project := methods.Cache.ProjectCache.GetProject(projectSlug)
	project.EnvVariables = []string{}
	methods.Cache.ProjectCache.SetProject(project)
	if methods.LsContext.Api.Token != "" {
		utils.GetAllProjectEnvVariables(methods.LsContext, methods.Cache, project.Slug)
	}
}
