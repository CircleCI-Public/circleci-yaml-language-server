package methods

import (
	"fmt"

	"github.com/segmentio/encoding/json"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
)

func (methods *Methods) References(reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	params := protocol.ReferenceParams{}
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(methods.Ctx, nil, fmt.Errorf("%s: %w", jsonrpc2.ErrParse, err))
	}

	res, err := languageservice.References(params, methods.Cache, methods.Settings)
	if err != nil {
		return reply(methods.Ctx, nil, err)
	}
	return reply(methods.Ctx, res, nil)
}
