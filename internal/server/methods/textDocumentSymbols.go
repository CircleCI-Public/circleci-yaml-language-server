package methods

import (
	"context"

	"go.lsp.dev/protocol"

	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
)

func (methods *Methods) DocumentSymbol(_ context.Context, params *protocol.DocumentSymbolParams) (protocol.DocumentSymbolResult, error) {
	res, err := languageservice.DocumentSymbols(*params, methods.Cache, methods.Settings)
	if err != nil {
		return nil, err
	}
	return protocol.DocumentSymbolSlice(res), nil
}
