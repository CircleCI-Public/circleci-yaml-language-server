package ast

import "go.lsp.dev/protocol"

// Function is a versioned binary declared in the top-level `functions` block,
// such as `setup-go: github.com/circleci-functions/setup-go@v0.1.0`. A step
// runs it by naming its alias, or one of its commands as `alias/command`.
type Function struct {
	Alias      string
	AliasRange protocol.Range
	// Reference is empty unless the value is a string.
	Reference      string
	IsString       bool
	ReferenceRange protocol.Range
	Range          protocol.Range
}
