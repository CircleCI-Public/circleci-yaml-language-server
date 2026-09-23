package expect

import (
	"testing"

	"go.lsp.dev/protocol"
)

type ExpectDiagnosticList struct {
	To To
}

type To struct {
	t    *testing.T
	list []protocol.Diagnostic
	Not  ToNot
	Be   ToBe
}

type ToNot struct {
	t    *testing.T
	list []protocol.Diagnostic

	Be ToNotBe
}

type ToNotBe struct {
	t    *testing.T
	list []protocol.Diagnostic
}

type ToBe struct {
	t    *testing.T
	list []protocol.Diagnostic
}

func DiagnosticList(t *testing.T, list []protocol.Diagnostic) ExpectDiagnosticList {
	return ExpectDiagnosticList{
		To: To{
			t:    t,
			list: list,

			Be: ToBe{
				t:    t,
				list: list,
			},

			Not: ToNot{
				t:    t,
				list: list,

				Be: ToNotBe{
					t:    t,
					list: list,
				},
			},
		},
	}
}

func (expect ToBe) Empty() {
	if len(expect.list) == 0 {
		return
	}

	listStr := diagnosticInfoList(expect.list, "\n\t")

	error := `Diagnostic list expected to be empty.
List:
%s
`

	expect.t.Errorf(error, listStr)
}

func (expect ToNotBe) Empty() {
	if len(expect.list) != 0 {
		return
	}

	expect.t.Error(`Diagnostic list expected to not be empty.`)
}

func (expect To) Include(
	diagnostic protocol.Diagnostic,
) {
	inList := ListHasDiagnostic(expect.list, diagnostic)

	if inList {
		return
	}

	listStr := diagnosticInfoList(expect.list, "\n\t")

	error := `Diagnostic not present in list
Expected: %s
List: %s
`
	expect.t.Errorf(error, "\n\t"+diagnosticInfo(diagnostic), listStr)
}

func (expect ToNot) Include(
	diagnostic protocol.Diagnostic,
) {
	inList := ListHasDiagnostic(expect.list, diagnostic)

	if !inList {
		return
	}

	listStr := diagnosticInfoList(expect.list, "\n\t")

	error := `Diagnostic present in list:%s`
	expect.t.Errorf(error, "\n\t"+diagnosticInfo(diagnostic), listStr)
}

func (expect To) IncludeAll(
	diagnostics []protocol.Diagnostic,
) {
	notFound := []protocol.Diagnostic{}

	for _, diagnostic := range diagnostics {
		inList := ListHasDiagnostic(expect.list, diagnostic)

		if inList {
			continue
		}

		notFound = append(notFound, diagnostic)
	}

	if len(notFound) == 0 {
		return
	}

	notFoundStr := diagnosticInfoList(notFound, "\n\t\t- ")
	listStr := diagnosticInfoList(expect.list, "\n\t\t- ")

	errorMessage := `Diagnostics not present in list
Not found: %s
List: %s`

	expect.t.Error(errorMessage, notFoundStr, listStr)
}

// Assert that the given val is not a complete subset of the list
func (expect ToNot) IncludeAll(
	val []protocol.Diagnostic,
) {
	found := []protocol.Diagnostic{}

	for _, diagnostic := range val {
		inList := ListHasDiagnostic(expect.list, diagnostic)

		if !inList {
			return
		}

		found = append(found, diagnostic)
	}

	notFoundStr := diagnosticInfoList(found, "\n\t\t- ")
	listStr := diagnosticInfoList(expect.list, "\n\t\t- ")

	errorMessage := `Diagnostics not present in list
Not found: %s
List: %s`

	expect.t.Error(errorMessage, notFoundStr, listStr)
}
