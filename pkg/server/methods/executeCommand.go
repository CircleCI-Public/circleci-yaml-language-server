package methods

import (
	"fmt"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"github.com/segmentio/encoding/json"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
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
		methods.updateAllCachedFiles()

	case "setSelfHostedUrl":
		param, ok := arguments[0].(string)
		if !ok {
			return reply(methods.Ctx, nil, nil)
		}
		methods.setHostUrl(param)
		methods.updateAllCachedFiles()

	case "setUserId":
		param, ok := arguments[0].(string)
		if !ok {
			return reply(methods.Ctx, nil, nil)
		}
		methods.setUserId(param)

	case "getWorkflows":
		content, okContent := arguments[0].(string)
		fileUri, okUri := arguments[1].(string)
		if !okContent || !okUri {
			return reply(methods.Ctx, nil, nil)
		}

		parsedFile, err := parser.ParseFromContent([]byte(content), methods.LsContext, uri.File(fileUri), protocol.Position{})
		if err != nil {
			return reply(methods.Ctx, nil, nil)
		}

		workflows := parsedFile.GetWorkflows()

		return reply(methods.Ctx, workflows, nil)
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
		go methods.notificationMethods(file.TextDocument)
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
		go methods.notificationMethods(file.TextDocument)
	}

	methods.updateProjectsEnvVariables()
}

func (methods *Methods) setUserId(userId string) {
	methods.LsContext.UserIdForTelemetry = userId
}
