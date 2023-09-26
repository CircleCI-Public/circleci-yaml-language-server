package validate

import (
	"sort"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/dockerhub"
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

func CreateValidateFromYAML(yaml string) Validate {
	context := testHelpers.GetDefaultLsContext()
	context.Api.Token = ""
	doc, _ := parser.ParseFromContent([]byte(yaml), context, uri.File(""), protocol.Position{})
	val := Validate{
		APIs: ValidateAPIs{
			DockerHub: dockerhub.NewAPI(),
		},
		Diagnostics: &[]protocol.Diagnostic{},
		Cache:       utils.CreateCache(),
		Doc:         doc,
		Context:     context,
	}
	return val
}

func CompareDiagnostics(t *testing.T, expected, actual *[]protocol.Diagnostic) {
	sortDiagnostic(expected)
	sortDiagnostic(actual)
	assert.Equal(t, expected, actual)
}

func CheckYamlErrors(t *testing.T, testCases []ValidateTestCase) {
	context := testHelpers.GetDefaultLsContext()
	context.Api.Token = ""
	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			val := CreateValidateFromYAML(tt.YamlContent)
			val.Validate(false)

			diags := *val.Diagnostics
			if tt.OnlyErrors == true {
				diags = getErrorDiagnostic(&diags)
			}

			if tt.Diagnostics == nil {
				assert.Len(t, diags, 0)
			} else {
				CompareDiagnostics(t, &tt.Diagnostics, &diags)
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
