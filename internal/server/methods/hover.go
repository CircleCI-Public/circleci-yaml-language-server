package methods

import (
	"context"

	"go.lsp.dev/protocol"

	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
)

func (methods *Methods) Hover(_ context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	res, err := languageservice.Hover(*params, methods.Cache, methods.Settings())
	if err != nil {
		return nil, nil
	}

	return &res, nil
}
