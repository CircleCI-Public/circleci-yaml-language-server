package methods

import (
	"fmt"

	languageservice "github.com/circleci/circleci-yaml-language-server/pkg/services"
	"github.com/segmentio/encoding/json"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

func (methods *Methods) Complete(reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	params := protocol.CompletionParams{}
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(methods.Ctx, nil, fmt.Errorf("%s: %w", jsonrpc2.ErrParse, err))
	}
	res, err := languageservice.Complete(params, methods.Cache)
	if err != nil {
		return reply(methods.Ctx, nil, err)
	}
	return reply(methods.Ctx, res, nil)
}
