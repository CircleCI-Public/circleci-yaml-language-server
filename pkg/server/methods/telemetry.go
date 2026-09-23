package methods

import (
	"log/slog"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/version"
	"go.lsp.dev/protocol"
)

type TelemetryEvent struct {
	Object      string                 `json:"object"`
	TriggerType string                 `json:"triggerType"`
	Action      string                 `json:"action"`
	Properties  map[string]interface{} `json:"properties"`
}

// TelemetryEvent are referenced on the following document:
// https://circleci.atlassian.net/wiki/spaces/DE/pages/6739722598/VS+Code+extension+Segment+event+tracking
// If you add an event in the code please edit the document
//
// If you don't know what to put in Action and TriggerType, leave them empty
// The lsp client may add other properties
func (methods *Methods) SendTelemetryEvent(event TelemetryEvent) {
	if event.Object == "" {
		event.Object = "lsp"
	}
	if event.TriggerType == "" {
		event.TriggerType = "frontend_interaction"
	}
	event.Properties["lspVersion"] = version.Server
	methods.notify(protocol.MethodTelemetryEvent, event)
}

// notify sends a notification to the client. There is no one to report a
// failure to, so it is logged.
func (methods *Methods) notify(method string, params any) {
	if err := methods.Conn.Notify(methods.Ctx, method, params); err != nil {
		slog.Warn("sending notification", "method", method, "err", err)
	}
}
