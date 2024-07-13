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
	testCases := []struct {
		label        string
		yamlData     string
		expectedDiag protocol.Diagnostic
	}{
		{
			label: "without default",
			yamlData: `jobs:
  test:
    parameters:
      os:
        type: executor
    executor: << parameters.os >>
    steps:
      - checkout`,
			expectedDiag: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 5, Character: 4},
					End:   protocol.Position{Line: 5, Character: 33},
				},
				Severity: protocol.DiagnosticSeverityWarning,
			},
		},
		{
			label: "with unknown default",
			yamlData: `jobs:
  test:
    parameters:
      os:
        type: executor
        default: unknown
    executor: << parameters.os >>
    steps:
      - checkout`,
			expectedDiag: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 6, Character: 4},
					End:   protocol.Position{Line: 6, Character: 33},
				},
				Severity: protocol.DiagnosticSeverityError,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run("executor parameter: "+testCase.label, func(t *testing.T) {
			ctx := &utils.LsContext{
				Api: utils.ApiContext{
					Token:   "XXXXXXXXXXXX",
					HostUrl: "https://circleci.com",
				},
			}
			doc, err := parser.ParseFromContent(
				[]byte(testCase.yamlData),
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

			for _, diag := range *val.Diagnostics {
				if diag.Range == testCase.expectedDiag.Range &&
					diag.Severity == testCase.expectedDiag.Severity {
					return
				}
			}
			t.Fatalf(`missing "parameter as executor" diagnostic`)
		})
	}
}

func TestResourceClass(t *testing.T) {
	testCases := []struct {
		label        string
		yamlData     string
		expectedDiag protocol.Diagnostic
	}{
		{
			label: "docker resource_class",
			yamlData: `jobs:
  test:
    docker:
      - image: ubuntu:latest
    resource_class: toto
    steps:
      - checkout`,
			expectedDiag: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 4, Character: 4},
					End:   protocol.Position{Line: 4, Character: 24},
				},
				Severity: protocol.DiagnosticSeverityError,
			},
		},
		{
			label: "docker resource_class",
			yamlData: `jobs:
  test:
    machine:
      image: ubuntu-2204:edge
    resource_class: toto
    steps:
      - checkout`,
			expectedDiag: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 4, Character: 4},
					End:   protocol.Position{Line: 4, Character: 24},
				},
				Severity: protocol.DiagnosticSeverityError,
			},
		},
		{
			label: "macos resource_class",
			yamlData: `jobs:
  test:
    macos:
      xcode: ` + utils.ValidXcodeVersions[0] + `
    resource_class: toto
    steps:
      - checkout`,
			expectedDiag: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 4, Character: 4},
					End:   protocol.Position{Line: 4, Character: 24},
				},
				Severity: protocol.DiagnosticSeverityError,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run("validate job resource_class: "+testCase.label, func(t *testing.T) {
			ctx := &utils.LsContext{
				Api: utils.ApiContext{
					Token:   "XXXXXXXXXXXX",
					HostUrl: "https://circleci.com",
				},
			}
			doc, err := parser.ParseFromContent(
				[]byte(testCase.yamlData),
				ctx,
				uri.URI(""),
				protocol.Position{},
			)
			assert.NoError(t, err, "invalid YAML data")
			assert.Contains(t, doc.Jobs, "test")

			val := Validate{
				APIs:        ValidateAPIs{DockerHubMock{}},
				Context:     ctx,
				Doc:         doc,
				Diagnostics: &[]protocol.Diagnostic{},
				Cache:       utils.CreateCache(),
			}
			val.validateSingleJob(doc.Jobs["test"])

			for _, diag := range *val.Diagnostics {
				if diag.Range == testCase.expectedDiag.Range &&
					diag.Severity == testCase.expectedDiag.Severity {
					return
				}
			}
			t.Fatalf(`missing resource_class diagnostic`)
		})
	}
}
