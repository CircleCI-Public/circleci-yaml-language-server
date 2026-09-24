package methods

import (
	"context"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/codeaction"
)

func (methods *Methods) CodeAction(_ context.Context, params *protocol.CodeActionParams) ([]protocol.CommandOrCodeAction, error) {
	res := []protocol.CommandOrCodeAction{}
	for _, diagnostic := range params.Context.Diagnostics {
		// The fixes for a diagnostic travel in its data, and come back here
		// when the client asks for them.
		codeActions, err := codeaction.FromData(diagnostic.Data)
		if err != nil {
			continue
		}
		for i := range codeActions {
			res = append(res, &codeActions[i])
		}
	}

	return res, nil
}
