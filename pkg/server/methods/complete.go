package methods

import (
	"fmt"

	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/services"
	"github.com/segmentio/encoding/json"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

func (methods *Methods) Complete(reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	params := protocol.CompletionParams{}

	reqParams := req.Params()
	err := json.Unmarshal(reqParams, &params)

	if err != nil {
		return reply(methods.Ctx, nil, fmt.Errorf("%s: %w", jsonrpc2.ErrParse, err))
	}

	res, err := languageservice.Complete(params, methods.Cache, methods.LsContext)
	go (func() {
		methods.SendTelemetryEvent(TelemetryEvent{
			Event:  "Complete",
			Action: "finished",
			Properties: DidOpenFinishedProperties{
				Filename: params.TextDocument.URI.Filename(),
			},
		})
	})()

	if err != nil {
		return reply(
			methods.Ctx,
			nil,
			err,
		)
	}

	return reply(methods.Ctx, res, nil)
}
