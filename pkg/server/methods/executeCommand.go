package methods

import (
	"fmt"

	"github.com/segmentio/encoding/json"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

func (methods *Methods) ExecuteCommand(reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	params := protocol.ExecuteCommandParams{}
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(methods.Ctx, nil, fmt.Errorf("%s: %w", jsonrpc2.ErrParse, err))
	}
	if params.Command == "setToken" {
		token := params.Arguments[0].(string)
		if methods.Cache.TokenCache.GetToken() != token {
			methods.Cache.RemoveOrbFiles()
			methods.Cache.OrbCache.RemoveOrbs()
		}
		methods.Cache.TokenCache.SetToken(token)
		filesCache := methods.Cache.FileCache.GetFiles()
		for _, file := range filesCache {
			go methods.notificationMethods(methods.Cache.FileCache, *file)
		}
	}
	if params.Command == "setSelfHostedUrl" {
		url := params.Arguments[0].(string)
		if methods.Cache.SelfHostedUrl.GetSelfHostedUrl() != url {
			methods.Cache.RemoveOrbFiles()
			methods.Cache.OrbCache.RemoveOrbs()
		}
		methods.Cache.SelfHostedUrl.SetUrl(url)
		filesCache := methods.Cache.FileCache.GetFiles()
		for _, file := range filesCache {
			go methods.notificationMethods(methods.Cache.FileCache, *file)
		}
	}
	return reply(methods.Ctx, nil, nil)
}
