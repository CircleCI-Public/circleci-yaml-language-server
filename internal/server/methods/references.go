package methods

import (
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
)

func (methods *Methods) References(raw jsonrpc2.RawMessage) (any, error) {
	params, err := decode[protocol.ReferenceParams](raw)
	if err != nil {
		return nil, err
	}

	res, err := languageservice.References(params, methods.Cache, methods.Settings)
	if err != nil {
		return nil, err
	}
	return res, nil
}
