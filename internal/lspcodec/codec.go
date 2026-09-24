// Package lspcodec is the jsonrpc2 codec for a connection that carries the
// protocol's types.
package lspcodec

import (
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

// Codec encodes payloads with the protocol's own codec, which is what reads
// and writes its union and optional fields. The protocol package has one too,
// but only wires it into the connections it builds itself.
type Codec struct{}

func (Codec) Marshal(v any) ([]byte, error) {
	if raw, ok := v.(jsonrpc2.RawMessage); ok {
		return raw, nil
	}
	return protocol.Marshal(v)
}

func (Codec) Unmarshal(data []byte, v any) error {
	return protocol.Unmarshal(data, v)
}
