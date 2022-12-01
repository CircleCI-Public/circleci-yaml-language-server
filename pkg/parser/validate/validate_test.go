package validate

import (
	"fmt"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
)

type ValidateTestCase struct {
	Name        string
	YamlContent string
	OnlyErrors  bool
	Diagnostics []protocol.Diagnostic
}

func CheckYamlErrors(t *testing.T, testCases []ValidateTestCase) {
	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			content := tt.YamlContent
			doc, err := parser.ParseFromContent([]byte(content))
			assert.Nil(t, err)
			val := Validate{
				Diagnostics: &[]protocol.Diagnostic{},
				Cache:       utils.CreateCache(),
				Doc:         doc,
			}
			val.Validate()

			diags := *val.Diagnostics
			if tt.OnlyErrors == true {
				diags = getErrorDiagnostic(&diags)
			}

			if tt.Diagnostics == nil {
				assert.Len(t, diags, 0)
			} else {
				fmt.Println(tt.Diagnostics)
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
