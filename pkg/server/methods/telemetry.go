package methods

import "go.lsp.dev/protocol"

type TelemetryEvent struct {
	Event       string                 `json:"event"`
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
	if event.Event == "" {
		event.Event = "vscode-lsp"
	}
	if event.TriggerType == "" {
		event.TriggerType = "frontend_interaction"
	}
	event.Properties["lspVersion"] = ServerVersion
	methods.Conn.Notify(methods.Ctx, protocol.MethodTelemetryEvent, event)
}
