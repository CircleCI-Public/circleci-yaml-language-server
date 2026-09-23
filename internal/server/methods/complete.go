package methods

import (
	"fmt"

	"github.com/segmentio/encoding/json"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
)

func (methods *Methods) Complete(reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	params := protocol.CompletionParams{}

	reqParams := req.Params()
	err := json.Unmarshal(reqParams, &params)

	if err != nil {
		return reply(methods.Ctx, nil, fmt.Errorf("%s: %w", jsonrpc2.ErrParse, err))
	}

	res, err := languageservice.Complete(params, methods.Cache, methods.Settings)
	go (func() {
		methods.SendTelemetryEvent(TelemetryEvent{
			Action: "autocompleted",
			Properties: map[string]interface{}{
				"filename": params.TextDocument.URI.Filename(),
			},
			TriggerType: "frontend_interaction",
			Object:      "lsp",
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
