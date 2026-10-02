package methods

import (
	"context"

	"go.lsp.dev/protocol"

	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
)

func (methods *Methods) DocumentLink(
	ctx context.Context, params *protocol.DocumentLinkParams,
) ([]protocol.DocumentLink, error) {
	return languageservice.DocumentLinks(ctx, *params, methods.Cache, methods.Settings())
}
