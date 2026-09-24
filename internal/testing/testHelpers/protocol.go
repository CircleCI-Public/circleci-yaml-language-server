package testHelpers

import (
	"slices"

	gocmp "github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.lsp.dev/protocol"
)

// ProtocolTypes lets go-cmp compare the protocol's types, whose optional
// values and diagnostic tags are held in unexported fields.
var ProtocolTypes = gocmp.Options{
	cmpopts.EquateComparable(
		protocol.Optional[string]{},
		protocol.Optional[bool]{},
		protocol.Optional[int32]{},
		protocol.Optional[uint32]{},
	),
	gocmp.Comparer(func(a, b protocol.DiagnosticTags) bool {
		return slices.Equal(a.Slice(), b.Slice())
	}),
}
