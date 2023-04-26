package validate

import (
	"sort"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/testHelpers"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type ValidateTestCase struct {
	Name        string
	YamlContent string
	// Whether you want to compare the Diagnostics to every diagnostics or only to the error diagnostics
	OnlyErrors  bool
	Diagnostics []protocol.Diagnostic
}

func CheckYamlErrors(t *testing.T, testCases []ValidateTestCase) {
	context := testHelpers.GetDefaultLsContext()
	context.Api.Token = ""
	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			content := tt.YamlContent
			doc, err := parser.ParseFromContent([]byte(content), context, uri.File(""), protocol.Position{})
			assert.Nil(t, err)
			val := Validate{
				Diagnostics: &[]protocol.Diagnostic{},
				Cache:       utils.CreateCache(),
				Doc:         doc,
				Context:     context,
			}
			val.Validate()

			diags := *val.Diagnostics
			if tt.OnlyErrors == true {
				diags = getErrorDiagnostic(&diags)
			}

			if tt.Diagnostics == nil {
				assert.Len(t, diags, 0)
			} else {
				sortDiagnostic(&diags)
				sortDiagnostic(&tt.Diagnostics)
				assert.Equal(t, tt.Diagnostics, diags)
			}
		})
	}
}

func getErrorDiagnostic(diags *[]protocol.Diagnostic) []protocol.Diagnostic {
	res := []protocol.Diagnostic{}
	for _, d := range *diags {
		if d.Severity == protocol.DiagnosticSeverityError {
			res = append(res, d)
		}
	}
	return res
}

func sortDiagnostic(diags *[]protocol.Diagnostic) {
	sort.Slice(*diags, func(i, j int) bool {
		if (*diags)[i].Range.Start.Line == (*diags)[j].Range.Start.Line {
			return (*diags)[i].Range.Start.Character < (*diags)[j].Range.Start.Character
		}
		if (*diags)[i].Range.End.Line == (*diags)[j].Range.End.Line {
			return (*diags)[i].Range.End.Character < (*diags)[j].Range.End.Character
		}

		return (*diags)[i].Range.Start.Line < (*diags)[j].Range.Start.Line
	})
}
