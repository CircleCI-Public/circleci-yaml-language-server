package expect

import (
	"fmt"
	"testing"

	"go.lsp.dev/protocol"
)

// Expect that all searched diagnostics are present in the
// in the given list.
// Diagnostics comparisons are performed using AreDiagnosticEqual
func ExpectAllDiagnosticInList(
	t *testing.T,
	completeList []protocol.Diagnostic,
	searchedDiagnostics []protocol.Diagnostic,
) {
	notFound := []protocol.Diagnostic{}

	for _, diagnostic := range searchedDiagnostics {
		inList := ListHasDiagnostic(completeList, diagnostic)

		if inList {
			continue
		}

		notFound = append(notFound, diagnostic)
	}

	if len(notFound) == 0 {
		return
	}

	notFoundStr := diagnosticInfoList(notFound, "\n\t\t- ")
	listStr := diagnosticInfoList(completeList, "\n\t\t- ")

	errorMessage := `Diagnostics not present in list
Not found: %s
List: %s`

	t.Error(errorMessage, notFoundStr, listStr)
}

// Expect that the given diagnostic is present in the given list.
// Diagnostic comparison is performed using AreDiagnosticEqual
func ExpectDiagnosticInList(t *testing.T, list []protocol.Diagnostic, diagnostic protocol.Diagnostic) {
	inList := ListHasDiagnostic(list, diagnostic)

	if inList {
		return
	}

	listStr := diagnosticInfoList(list, "\n\t")

	error := `Diagnostic not present in list
Expected: %s
List: %s
`
	t.Errorf(error, "\n\t"+diagnosticInfo(diagnostic), listStr)
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
