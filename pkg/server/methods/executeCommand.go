package methods

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"github.com/rollbar/rollbar-go"
	"github.com/segmentio/encoding/json"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func (methods *Methods) ExecuteCommand(reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	params := protocol.ExecuteCommandParams{}
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(methods.Ctx, nil, jsonrpc2.NewError(jsonrpc2.ParseError, err.Error()))
	}

	arguments := params.Arguments

	switch params.Command {
	case "setToken":
		param, ok := arguments[0].(string)
		if !ok {
			return reply(methods.Ctx, nil, jsonrpc2.NewError(jsonrpc2.InvalidParams, "invalid method parameter: token"))
		}
		methods.setToken(param)
		methods.updateAllCachedFiles()

	case "setSelfHostedUrl":
		param, ok := arguments[0].(string)
		if !ok {
			return reply(methods.Ctx, nil, jsonrpc2.NewError(jsonrpc2.InvalidParams, "invalid method parameter: selfHostedURL"))
		}
		methods.setHostUrl(param)
		methods.updateAllCachedFiles()

	case "setUserId":
		param, ok := arguments[0].(string)
		if !ok {
			return reply(methods.Ctx, nil, jsonrpc2.NewError(jsonrpc2.InvalidParams, "invalid method parameter: userId"))
		}
		methods.setUserId(param)

	case "getWorkflows":
		content, okContent := arguments[0].(string)
		if !okContent {
			return reply(methods.Ctx, nil, jsonrpc2.NewError(jsonrpc2.InvalidParams, "invalid method parameter: fileContent"))
		}
		fileUri, okUri := arguments[1].(string)
		if !okUri {
			return reply(methods.Ctx, nil, jsonrpc2.NewError(jsonrpc2.InvalidParams, "invalid method parameter: fileURI"))
		}

		parsedFile, err := parser.ParseFromContent([]byte(content), methods.LsContext, uri.File(fileUri), protocol.Position{})
		if err != nil {
			return reply(methods.Ctx, nil, jsonrpc2.NewError(jsonrpc2.InternalError, "unable to parse file"))
		}

		workflows := parsedFile.GetWorkflows()

		return reply(methods.Ctx, workflows, nil)

	case "setRollbarInformation":
		parameters, ok := arguments[0].(map[string]interface{})
		if !ok {
			return reply(methods.Ctx, nil, jsonrpc2.NewError(jsonrpc2.InvalidParams, "invalid method parameter: parameters"))
		}

		for key, value := range parameters {
			switch key {
			case "enabled":
				if enabled, ok := value.(bool); ok {
					rollbar.SetEnabled(enabled)
					delete(parameters, key)
				}
			case "environment":
				if env, ok := value.(string); ok {
					rollbar.SetEnvironment(env)
					delete(parameters, key)
				}
			case "personId":
				if personId, ok := value.(string); ok {
					rollbar.SetPerson(personId, "", "")
					delete(parameters, key)
				}
			}
		}
		rollbar.SetCustom(parameters)
	}

	return reply(methods.Ctx, &struct{}{}, nil)
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
