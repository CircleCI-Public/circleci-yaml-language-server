package validate

import (
	"fmt"
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

func TestRetention(t *testing.T) {
	testCases := []struct {
		label        string
		yamlData     string
		expectedDiag protocol.Diagnostic
	}{
		{
			label: "invalid retention caches - too high",
			yamlData: `jobs:
  test:
    retention:
      caches: 16d
    steps:
      - checkout`,
			expectedDiag: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 3, Character: 6},
					End:   protocol.Position{Line: 3, Character: 17},
				},
				Severity: protocol.DiagnosticSeverityError,
			},
		},
		{
			label: "invalid retention caches - too low",
			yamlData: `jobs:
  test:
    retention:
      caches: 0d
    steps:
      - checkout`,
			expectedDiag: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 3, Character: 6},
					End:   protocol.Position{Line: 3, Character: 16},
				},
				Severity: protocol.DiagnosticSeverityError,
			},
		},
		{
			label: "invalid retention caches - invalid format",
			yamlData: `jobs:
  test:
    retention:
      caches: abc
    steps:
      - checkout`,
			expectedDiag: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 3, Character: 6},
					End:   protocol.Position{Line: 3, Character: 17},
				},
				Severity: protocol.DiagnosticSeverityError,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run("validate job retention: "+testCase.label, func(t *testing.T) {
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
			t.Fatalf(`missing retention diagnostic for test case: %s`, testCase.label)
		})
	}
}

func TestJobTypeValidation(t *testing.T) {
	// WARNING: be careful when editing the `yamlData` strings as they are sensitive to tabs vs spaces

	testCases := []struct {
		label        string
		yamlData     string
		expectedDiag protocol.Diagnostic
	}{
		{
			label: "explicitly defining type as build gives a hint",
			yamlData: `jobs:
  my-job:
    type: build
`,
			expectedDiag: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 2, Character: 4},
					End:   protocol.Position{Line: 2, Character: 15},
				},
				Severity: protocol.DiagnosticSeverityHint,
				Message:  "If no `type:` key is specified, the job will default to `type: build`.",
			},
		},
		{
			label: "invalid job type should give an error",
			yamlData: `jobs:
  my-job:
    type: bad-type
`,
			expectedDiag: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 2, Character: 4},
					End:   protocol.Position{Line: 2, Character: 18},
				},
				Severity: protocol.DiagnosticSeverityError,
				Message:  "Invalid job type 'bad-type'. Allowed types: approval, build, no-op, release, lock, unlock",
			},
		},
		{
			label: "putting `steps`: in a job type that doesn't use it will give a warning",
			yamlData: `jobs:
  my-job:
    type: approval
    steps:
      - checkout`,
			expectedDiag: protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: 3, Character: 4},
					End:   protocol.Position{Line: 4, Character: 16},
				},
				Severity: protocol.DiagnosticSeverityWarning,
				Message:  "Steps only exist in `build` jobs. Steps here will be ignored.",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run("validate job type: "+testCase.label, func(t *testing.T) {
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

			val := Validate{
				APIs:        ValidateAPIs{DockerHubMock{}},
				Context:     ctx,
				Doc:         doc,
				Diagnostics: &[]protocol.Diagnostic{},
				Cache:       utils.CreateCache(),
			}

			val.Validate()

			diagnostics := ""
			for _, diag := range *val.Diagnostics {
				formattedDiag := fmt.Sprintf("%v\n", diag)
				diagnostics = diagnostics + formattedDiag
				if diag.Range == testCase.expectedDiag.Range &&
					diag.Severity == testCase.expectedDiag.Severity &&
					diag.Message == testCase.expectedDiag.Message {
					return
				}
			}

			t.Fatalf("missing type validation diagnostic for test case: %s\n\nEmitted diagnostic:\n%v\nExpected diagnostic:\n%v", testCase.label, diagnostics, testCase.expectedDiag)
		})
	}
}
