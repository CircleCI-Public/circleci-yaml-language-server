package methods

import (
	"encoding/json"
	"fmt"
	lsp "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/services"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

func (methods *Methods) SemanticTokens(reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	params := protocol.SemanticTokensParams{}
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(methods.Ctx, nil, fmt.Errorf("%s: %w", jsonrpc2.ErrParse, err))
	}

	res := lsp.SemanticTokens(params, methods.Cache)

	return reply(methods.Ctx, res, nil)
}
