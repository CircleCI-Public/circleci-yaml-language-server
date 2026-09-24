package methods

import (
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/codeaction"
)

func (methods *Methods) CodeAction(raw jsonrpc2.RawMessage) (any, error) {
	params, err := decode[protocol.CodeActionParams](raw)
	if err != nil {
		return nil, err
	}

	res := []protocol.CodeAction{}
	for _, diagnostic := range params.Context.Diagnostics {
		// The fixes for a diagnostic travel in its data, and come back here
		// when the client asks for them.
		codeActions, err := codeaction.FromData(diagnostic.Data)
		if err != nil {
			continue
		}
		res = append(res, codeActions...)
	}

	return res, nil
}
