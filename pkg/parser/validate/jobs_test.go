package validate

import (
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestExecutorParam(t *testing.T) {
	yamlData := `jobs:
  test:
    parameters:
      os:
        type: executor
    executor: << parameters.os >>
    steps:
      - checkout`
	ctx := &utils.LsContext{
		Api: utils.ApiContext{
			Token:   "XXXXXXXXXXXX",
			HostUrl: "https://circleci.com",
		},
	}
	doc, err := parser.ParseFromContent(
		[]byte(yamlData),
		ctx,
		uri.URI(""),
		protocol.Position{},
	)
	assert.NoError(t, err, "invalid YAML data")
	assert.Contains(t, doc.Jobs, "test")

	val := Validate{
		Context:     ctx,
		Doc:         doc,
		Diagnostics: &[]protocol.Diagnostic{},
	}
	val.validateSingleJob(doc.Jobs["test"])

	expectedDiag := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 5, Character: 4},
			End:   protocol.Position{Line: 5, Character: 33},
		},
		Severity: protocol.DiagnosticSeverityError,
	}
	for _, diag := range *val.Diagnostics {
		if diag.Range == expectedDiag.Range && diag.Severity == expectedDiag.Severity {
			return
		}
	}
	t.Fatalf(`missing "parameter as executor" diagnostic`)
}
