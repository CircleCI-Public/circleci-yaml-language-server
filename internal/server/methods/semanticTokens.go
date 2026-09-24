package methods

import (
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	lsp "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
)

func (methods *Methods) SemanticTokens(raw jsonrpc2.RawMessage) (any, error) {
	params, err := decode[protocol.SemanticTokensParams](raw)
	if err != nil {
		return nil, err
	}

	return lsp.SemanticTokens(params, methods.Cache, methods.Settings), nil
}
