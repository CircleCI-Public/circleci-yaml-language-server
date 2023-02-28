package ast

import "go.lsp.dev/protocol"

type TextAndRange struct {
	Text  string
	Range protocol.Range
}
