package methods

import (
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
)

func (methods *Methods) Complete(raw jsonrpc2.RawMessage) (any, error) {
	params, err := decode[protocol.CompletionParams](raw)
	if err != nil {
		return nil, err
	}

	res, err := languageservice.Complete(params, methods.Cache, methods.Settings)
	go (func() {
		methods.SendTelemetryEvent(TelemetryEvent{
			Action: "autocompleted",
			Properties: map[string]interface{}{
				"filename": params.TextDocument.URI.FsPath(),
			},
			TriggerType: "frontend_interaction",
			Object:      "lsp",
		})
	})()

	if err != nil {
		return nil, err
	}

	return res, nil
}
