package ast

import (
	"go.lsp.dev/protocol"
)

type Environment struct {
	Range protocol.Range
	Keys  []string
}
