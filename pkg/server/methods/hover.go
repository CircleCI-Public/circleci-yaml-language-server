package methods

import (
	"fmt"

	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/services"
	"github.com/segmentio/encoding/json"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

func (methods *Methods) Hover(reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	params := protocol.HoverParams{}
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(methods.Ctx, nil, fmt.Errorf("%s: %w", jsonrpc2.ErrParse, err))
	}

	res, err := languageservice.Hover(params, methods.Cache, methods.LsContext)
	if err != nil {
		return reply(methods.Ctx, nil, nil)
	}

	return reply(methods.Ctx, res, nil)
}
