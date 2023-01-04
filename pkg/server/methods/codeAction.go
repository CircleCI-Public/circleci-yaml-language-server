package methods

import (
	"fmt"

	"github.com/segmentio/encoding/json"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

func (methods *Methods) CodeAction(reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	params := protocol.CodeActionParams{}
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(methods.Ctx, nil, fmt.Errorf("%s: %w", jsonrpc2.ErrParse, err))
	}

	res := []protocol.CodeAction{}
	for _, diagnostic := range params.Context.Diagnostics {
		if diagnostic.Data == nil {
			continue
		}

		// We do this because the type of diagnostic.Data is map[string]interface{}
		// and we need to convert it to []protocol.CodeAction
		str, _ := json.Marshal(diagnostic.Data)
		codeActions := []protocol.CodeAction{}
		err := json.Unmarshal(str, &codeActions)
		if err != nil {
			continue
		}
		res = append(res, codeActions...)
	}

	return reply(methods.Ctx, res, nil)
}
