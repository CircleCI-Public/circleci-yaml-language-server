package methods

import (
	"context"

	"go.lsp.dev/protocol"

	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
)

func (methods *Methods) DocumentHighlight(
	_ context.Context, params *protocol.DocumentHighlightParams,
) ([]protocol.DocumentHighlight, error) {
	return languageservice.DocumentHighlight(context.TODO(), *params, methods.Cache, methods.Settings())
}
