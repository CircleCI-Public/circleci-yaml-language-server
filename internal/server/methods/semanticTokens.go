package methods

import (
	"context"

	"go.lsp.dev/protocol"

	lsp "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
)

func (methods *Methods) SemanticTokensFull(_ context.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	tokens := lsp.SemanticTokens(*params, methods.Cache, methods.Settings())
	return &tokens, nil
}
