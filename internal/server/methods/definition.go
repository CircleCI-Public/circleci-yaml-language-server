package methods

import (
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
)

func (methods *Methods) Definition(raw jsonrpc2.RawMessage) (any, error) {
	params, err := decode[protocol.DefinitionParams](raw)
	if err != nil {
		return nil, err
	}

	res, err := languageservice.Definition(params, methods.Cache, methods.Settings)
	if err != nil {
		return nil, err
	}
	// Nothing found has always been answered with null; encoded as it is, the
	// nil slice would go out as [].
	if res == nil {
		return nil, nil
	}
	return res, nil
}
