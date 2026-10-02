package methods

import (
	"context"

	"go.lsp.dev/protocol"

	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
)

func (methods *Methods) DocumentHighlight(
	ctx context.Context, params *protocol.DocumentHighlightParams,
) ([]protocol.DocumentHighlight, error) {
	return languageservice.DocumentHighlight(ctx, *params, methods.Cache, methods.Settings())
}
