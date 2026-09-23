package expect

import (
	"testing"

	"go.lsp.dev/protocol"
)

type Diag struct {
	t          *testing.T
	diagnostic protocol.Diagnostic

	To DiagTo
}

type DiagTo struct {
	t          *testing.T
	diagnostic protocol.Diagnostic

	Not DiagToNot
}

type DiagToNot struct {
	t          *testing.T
	diagnostic protocol.Diagnostic
}

func Diagnostic(t *testing.T, diagnostic protocol.Diagnostic) Diag {
	return Diag{
		t:          t,
		diagnostic: diagnostic,

		To: DiagTo{
			t:          t,
			diagnostic: diagnostic,
		},
	}
}

func (expect DiagTo) Equal(diagnostic protocol.Diagnostic) {
	if AreDiagnosticEquivalent(expect.diagnostic, diagnostic) {
		return
	}

	message := `Expecting diagnostics to be equal.
Expected: %s
Actual: %s
`

	expect.t.Errorf(
		message,
		diagnosticInfo(diagnostic),
		diagnosticInfo(expect.diagnostic),
	)
}

func (expect DiagToNot) Equal(diagnostic protocol.Diagnostic) {
	if !AreDiagnosticEquivalent(expect.diagnostic, diagnostic) {
		return
	}

	message := `Expecting diagnostics to be different.
Expected: %s
Actual: %s
`

	expect.t.Errorf(
		message,
		diagnosticInfo(diagnostic),
		diagnosticInfo(expect.diagnostic),
	)
}
