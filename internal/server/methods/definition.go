package methods

import (
	"context"

	"go.lsp.dev/protocol"

	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
)

func (methods *Methods) Definition(_ context.Context, params *protocol.DefinitionParams) (protocol.DefinitionResult, error) {
	res, err := languageservice.Definition(*params, methods.Cache, methods.Settings)
	if err != nil {
		return nil, err
	}
	// Nothing found has always been answered with null; as a slice, even a
	// nil one would go out as [].
	if res == nil {
		return nil, nil
	}
	return protocol.LocationSlice(res), nil
}
