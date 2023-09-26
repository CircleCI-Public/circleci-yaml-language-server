package expect

import (
	"fmt"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
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

// Return a string displaying a list of diagnostic information.
func diagnosticInfoList(list []protocol.Diagnostic, prefix string) string {
	listStr := ""
	for _, diag := range list {
		listStr += prefix + diagnosticInfo(diag)
	}

	return listStr
}

// Return a string containing the diagnostic information.
func diagnosticInfo(diagnostic protocol.Diagnostic) string {
	return fmt.Sprintf(
		"<L%d:%d,L%d:%d> %s: %s",
		diagnostic.Range.Start.Line,
		diagnostic.Range.Start.Character,
		diagnostic.Range.End.Line,
		diagnostic.Range.End.Character,
		diagnostic.Severity,
		diagnostic.Message,
	)
}
