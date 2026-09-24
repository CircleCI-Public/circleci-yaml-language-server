package methods

import (
	"context"

	"go.lsp.dev/protocol"

	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
)

func (methods *Methods) References(_ context.Context, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	return languageservice.References(*params, methods.Cache, methods.Settings())
}
