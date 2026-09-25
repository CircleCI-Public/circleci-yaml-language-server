package methods

import (
	"context"

	"github.com/rollbar/rollbar-go"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

func (methods *Methods) ExecuteCommand(_ context.Context, params *protocol.ExecuteCommandParams) (protocol.LSPAny, error) {
	arguments := params.Arguments

	switch params.Command {
	case "setToken":
		param, ok := argument[string](arguments, 0)
		if !ok {
			return nil, jsonrpc2.NewError(jsonrpc2.InvalidParams, "invalid method parameter: token")
		}
		methods.setToken(param)
		methods.updateAllCachedFiles()

	case "setGitHubToken":
		param, ok := argument[string](arguments, 0)
		if !ok {
			return nil, jsonrpc2.NewError(jsonrpc2.InvalidParams, "invalid method parameter: token")
		}
		methods.setGitHubToken(param)

	case "setSelfHostedUrl":
		param, ok := argument[string](arguments, 0)
		if !ok {
			return nil, jsonrpc2.NewError(jsonrpc2.InvalidParams, "invalid method parameter: selfHostedURL")
		}
		methods.setHostUrl(param)
		methods.updateAllCachedFiles()

	case "setUserId":
		param, ok := argument[string](arguments, 0)
		if !ok {
			return nil, jsonrpc2.NewError(jsonrpc2.InvalidParams, "invalid method parameter: userId")
		}
		methods.setUserId(param)

	case "getWorkflows":
		content, okContent := argument[string](arguments, 0)
		if !okContent {
			return nil, jsonrpc2.NewError(jsonrpc2.InvalidParams, "invalid method parameter: fileContent")
		}
		fileUri, okUri := argument[string](arguments, 1)
		if !okUri {
			return nil, jsonrpc2.NewError(jsonrpc2.InvalidParams, "invalid method parameter: fileURI")
		}

		parsedFile, err := parser.ParseFromContent([]byte(content), methods.Settings(), uri.File(fileUri), protocol.Position{})
		if err != nil {
			return nil, jsonrpc2.NewError(jsonrpc2.InternalError, "unable to parse file")
		}
		defer parsedFile.Close()

		workflows, err := protocol.Marshal(parsedFile.GetWorkflows())
		if err != nil {
			return nil, jsonrpc2.NewError(jsonrpc2.InternalError, "unable to encode workflows")
		}
		return workflows, nil

	case "setRollbarInformation":
		parameters, ok := argument[map[string]interface{}](arguments, 0)
		if !ok {
			return nil, jsonrpc2.NewError(jsonrpc2.InvalidParams, "invalid method parameter: parameters")
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

	return nil, nil
}

// argument is a command's argument at i, and whether there is one of type T
// there.
func argument[T any](arguments []protocol.LSPAny, i int) (T, bool) {
	var value T
	if i >= len(arguments) {
		return value, false
	}
	if err := protocol.Unmarshal(arguments[i], &value); err != nil {
		return value, false
	}
	return value, true
}

func (methods *Methods) setToken(token string) {
	if methods.Settings().Api.Token != token {
		methods.Cache.ClearHostData()
	}

	methods.updateSettings(func(settings *session.Settings) {
		settings.Api.Token = token
	})
	filesCache := methods.Cache.FileCache.GetFiles()
	for _, file := range filesCache {
		go methods.notificationMethods(file.TextDocument)
	}

	methods.updateProjectsEnvVariables()
}

// setGitHubToken sets the token orbs referenced by a GitHub URL are fetched
// with, when they can't be without one. An orb fetched, or found missing,
// with the old token is forgotten.
func (methods *Methods) setGitHubToken(token string) {
	if methods.Settings().OrbURLs.GitHubToken == token {
		return
	}

	methods.Cache.OrbCache.Clear()
	methods.updateSettings(func(settings *session.Settings) {
		settings.OrbURLs.GitHubToken = token
	})

	for _, file := range methods.Cache.FileCache.GetFiles() {
		go methods.notificationMethods(file.TextDocument)
	}
}

func (methods *Methods) setHostUrl(hostUrl string) {
	if methods.Settings().Api.HostUrl != hostUrl {
		methods.Cache.ClearHostData()
	}

	if hostUrl == "" {
		hostUrl = circleci.DefaultHostURL
	}
	methods.updateSettings(func(settings *session.Settings) {
		settings.Api.HostUrl = hostUrl
	})

	filesCache := methods.Cache.FileCache.GetFiles()
	for _, file := range filesCache {
		go methods.notificationMethods(file.TextDocument)
	}

	methods.updateProjectsEnvVariables()
}

func (methods *Methods) setUserId(userId string) {
	methods.updateSettings(func(settings *session.Settings) {
		settings.UserIdForTelemetry = userId
	})
}
