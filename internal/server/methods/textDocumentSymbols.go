package methods

import (
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
)

func (methods *Methods) DocumentSymbols(raw jsonrpc2.RawMessage) (any, error) {
	params, err := decode[protocol.DocumentSymbolParams](raw)
	if err != nil {
		return nil, err
	}

	return languageservice.DocumentSymbols(params, methods.Cache, methods.Settings)
}
