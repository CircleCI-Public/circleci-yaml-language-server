package methods

import (
	"context"

	"go.lsp.dev/protocol"

	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
)

func (methods *Methods) References(ctx context.Context, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	return languageservice.References(ctx, *params, methods.Cache, methods.Settings())
}
