package ast

import (
	"go.lsp.dev/protocol"
)

type Environment struct {
	Range     protocol.Range
	Variables []EnvironmentVariable
}

// EnvironmentVariable is one key of an environment map, with the range of
// its whole `key: value` pair and of the key alone.
type EnvironmentVariable struct {
	Name      string
	Range     protocol.Range
	NameRange protocol.Range
}
