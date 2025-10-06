package parser

import (
	"strings"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/expect"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/testHelpers"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gopkg.in/yaml.v3"
)

func Test_HandleYAMLErrors_MappingKeyError(t *testing.T) {
	content := []byte(`
anchorA: &anchorA
  A: 1

anchorB: &anchorB
  B: 2

anchorC: &anchorC
  C: 3

testFinal:
  <<: *anchorA
  <<: *anchorB
  <<: *anchorC
`)

	m := make(map[interface{}]interface{})

	err := yaml.Unmarshal(content, m)

	context := testHelpers.GetDefaultLsContext()
	yamlDocument, _ := ParseFromContent(content, context, uri.File(""), protocol.Position{})

	actualDiagnostics, err := handleYAMLErrors(err.Error(), content, yamlDocument.RootNode)

	assert.Nil(t, err)

	expectedDiagnostics := []protocol.Diagnostic{}

	expect.DiagnosticList(t, actualDiagnostics).To.IncludeAll(expectedDiagnostics)
}

func Test_HandleYamlError_UnknownAnchor(t *testing.T) {
	content := []byte(`
test:
  <<: *unknownAnchor
`)

	m := make(map[interface{}]interface{})

	err := yaml.Unmarshal(content, m)

	context := testHelpers.GetDefaultLsContext()
	yamlDocument, _ := ParseFromContent(content, context, uri.File(""), protocol.Position{})

	diagnostics, err := handleYAMLErrors(err.Error(), content, yamlDocument.RootNode)

	assert.Nil(t, err)

	expected := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 2, Character: 7},
			End:   protocol.Position{Line: 2, Character: 20},
		},
		Severity: protocol.DiagnosticSeverityError,
		Message:  "yaml: unknown anchor 'unknownAnchor' referenced",
	}

	expect.DiagnosticList(t, diagnostics).To.Include(expected)
}

func Test_JobDefinitionTypes(t *testing.T) {
	testCases := []struct {
		name                string
		yaml                string
		expectError         bool
		expectErrorContains string
	}{
		// Build type tests
		{
			name: "build type - valid with docker and steps",
			yaml: `
version: 2.1
jobs:
  my-build-job:
    type: build
    docker:
      - image: cimg/base:2023.01
    steps:
      - checkout
`,
			expectError: false,
		},
		{
			name: "build type - valid with explicit type",
			yaml: `
version: 2.1
jobs:
  my-build-job:
    type: build
    docker:
      - image: cimg/base:2023.01
    steps:
      - checkout
      - run: echo "build job with explicit type"
`,
			expectError: false,
		},
		{
			name: "build type - missing steps (should error)",
			yaml: `
version: 2.1
jobs:
  my-build-job:
    type: build
    docker:
      - image: cimg/base:2023.01
`,
			expectError:         true,
			expectErrorContains: "steps",
		},

		// Release type tests
		{
			name: "release type - valid with plan_name",
			yaml: `
version: 2.1
jobs:
  my-release-job:
    type: release
    plan_name: my-release-plan
`,
			expectError: false,
		},
		{
			name: "release type - valid with additional properties",
			yaml: `
version: 2.1
jobs:
  my-release-job:
    type: release
    plan_name: my-plan
    some_other_property: allowed
`,
			expectError: false,
		},
		{
			name: "release type - missing plan_name (should error)",
			yaml: `
version: 2.1
jobs:
  my-release-job:
    type: release
`,
			expectError:         true,
			expectErrorContains: "plan_name",
		},

		// Lock type tests
		{
			name: "lock type - valid with key",
			yaml: `
version: 2.1
jobs:
  my-lock-job:
    type: lock
    key: my-lock-key
`,
			expectError: false,
		},
		{
			name: "lock type - valid with additional properties",
			yaml: `
version: 2.1
jobs:
  my-lock-job:
    type: lock
    key: my-key
    some_other_property: allowed
`,
			expectError: false,
		},
		{
			name: "lock type - missing key (should error)",
			yaml: `
version: 2.1
jobs:
  my-lock-job:
    type: lock
`,
			expectError:         true,
			expectErrorContains: "key",
		},

		// Unlock type tests
		{
			name: "unlock type - valid with key",
			yaml: `
version: 2.1
jobs:
  my-unlock-job:
    type: unlock
    key: my-lock-key
`,
			expectError: false,
		},
		{
			name: "unlock type - valid with additional properties",
			yaml: `
version: 2.1
jobs:
  my-unlock-job:
    type: unlock
    key: my-key
    some_other_property: allowed
`,
			expectError: false,
		},
		{
			name: "unlock type - missing key (should error)",
			yaml: `
version: 2.1
jobs:
  my-unlock-job:
    type: unlock
`,
			expectError:         true,
			expectErrorContains: "key",
		},

		// Approval type tests
		{
			name: "approval type - valid minimal",
			yaml: `
version: 2.1
jobs:
  my-approval-job:
    type: approval
`,
			expectError: false,
		},
		{
			name: "approval type - valid with steps (ignored)",
			yaml: `
version: 2.1
jobs:
  my-approval-job:
    type: approval
    steps:
      - run: echo "This will be ignored"
`,
			expectError: false,
		},

		// No-op type tests
		{
			name: "no-op type - valid minimal",
			yaml: `
version: 2.1
jobs:
  my-noop-job:
    type: no-op
`,
			expectError: false,
		},
		{
			name: "no-op type - valid with steps (ignored)",
			yaml: `
version: 2.1
jobs:
  my-noop-job:
    type: no-op
    steps:
      - run: echo "This will be ignored"
`,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			context := testHelpers.GetDefaultLsContext()
			yamlDocument, _ := ParseFromContent([]byte(tc.yaml), context, uri.File(""), protocol.Position{})

			if tc.expectError {
				// For error cases, also run JSON schema validation
				validator := JSONSchemaValidator{
					Doc: yamlDocument,
				}

				schemaPath := "../../schema.json"
				err := validator.LoadJsonSchema(schemaPath)
				if err != nil {
					t.Logf("Warning: Could not load schema: %v", err)
					t.SkipNow()
				}

				diagnostics := validator.ValidateWithJSONSchema(yamlDocument.RootNode, yamlDocument.Content)

				// Log all diagnostics for debugging
				if len(diagnostics) > 0 {
					t.Logf("Found %d diagnostic(s):", len(diagnostics))
					for _, d := range diagnostics {
						t.Logf("  - %s", d.Message)
					}
				} else {
					t.Logf("No diagnostics found")
				}

				assert.NotEmpty(t, diagnostics, "Expected validation errors but got none")
				if tc.expectErrorContains != "" {
					found := false
					for _, d := range diagnostics {
						if strings.Contains(strings.ToLower(d.Message), strings.ToLower(tc.expectErrorContains)) {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected error message to contain '%s'", tc.expectErrorContains)
				}
			} else {
				// For non-error cases, just check that parsing succeeded
				diagnostics := yamlDocument.Diagnostics
				assert.Empty(t, diagnostics, "Expected no errors but got: %v", diagnostics)
			}
		})
	}
}
