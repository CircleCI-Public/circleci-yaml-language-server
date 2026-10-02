package methods

import (
	"context"

	"go.lsp.dev/protocol"

	lsp "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
)

func (methods *Methods) SemanticTokensFull(ctx context.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	tokens := lsp.SemanticTokens(ctx, *params, methods.Cache, methods.Settings())
	return &tokens, nil
}
