package methods

import (
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
)

func (methods *Methods) Hover(raw jsonrpc2.RawMessage) (any, error) {
	params, err := decode[protocol.HoverParams](raw)
	if err != nil {
		return nil, err
	}

	res, err := languageservice.Hover(params, methods.Cache, methods.Settings)
	if err != nil {
		return nil, nil
	}

	return res, nil
}
