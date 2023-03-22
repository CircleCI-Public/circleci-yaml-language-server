package ast

import "go.lsp.dev/protocol"

type TextAndRange struct {
	Text  string         `json:"text"`
	Range protocol.Range `json:"range"`
}
