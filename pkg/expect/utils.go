package expect

import (
	"github.com/circleci/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

// Indicate if a diagnostic is present in the given list
// Will return true if a diagnostic match all the given conditions:
//   - Same code
//   - Same message
//   - Same range
//   - Same severity
func ListHasDiagnostic(list []protocol.Diagnostic, diagnostic protocol.Diagnostic) bool {
	if len(list) == 0 {
		return false
	}

	for _, diag := range list {
		if AreDiagnosticEquivalent(diag, diagnostic) {
			return true
		}
	}

	return false
}

// Compare two diagnostics and indicate if they are equivalents or not
// Two diagnostics are equivalent if they match all of the following conditions:
//   - Same code
//   - Same message
//   - Same range
//   - Same severity
func AreDiagnosticEquivalent(a protocol.Diagnostic, b protocol.Diagnostic) bool {
	if a.Severity != b.Severity {
		return false
	}

	if a.Message != b.Message {
		return false
	}

	if a.Code != b.Code {
		return false
	}

	return utils.AreRangeEqual(a.Range, b.Range)
}
