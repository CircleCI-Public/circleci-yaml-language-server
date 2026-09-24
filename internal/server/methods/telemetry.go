package methods

import (
	"log/slog"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/version"
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

	encoded, err := protocol.Marshal(event)
	if err != nil {
		slog.Warn("encoding notification", "method", protocol.MethodTelemetryEvent, "err", err)
		return
	}
	if err := methods.Client.Telemetry(methods.Ctx, encoded); err != nil {
		slog.Warn("sending notification", "method", protocol.MethodTelemetryEvent, "err", err)
	}
}

// publishDiagnostics and the telemetry below send notifications to the
// client. There is no one to report a failure to, so it is logged.
func (methods *Methods) publishDiagnostics(params protocol.PublishDiagnosticsParams) {
	if err := methods.Client.PublishDiagnostics(methods.Ctx, &params); err != nil {
		slog.Warn("sending notification", "method", protocol.MethodTextDocumentPublishDiagnostics, "err", err)
	}
}
