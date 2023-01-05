package methods

import "go.lsp.dev/protocol"

type TelemetryEvent struct {
	Event      string `json:"event"`
	Action     string `json:"action"`
	Properties interface{}
}

type DidOpenFinishedProperties struct {
	Filename string `json:"filename"`
}

func (methods *Methods) SendTelemetryEvent(event TelemetryEvent) {
	methods.Conn.Notify(methods.Ctx, protocol.MethodTelemetryEvent, event)
}
