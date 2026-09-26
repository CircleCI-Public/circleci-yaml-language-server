package parser

import (
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gopkg.in/yaml.v3"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	schema "github.com/CircleCI-Public/circleci-yaml-language-server"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/expect"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
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

	context := testHelpers.DefaultSettings()
	yamlDocument, _ := ParseFromContent(content, context, uri.File(""), protocol.Position{})

	actualDiagnostics, err := handleYAMLErrors(err.Error(), content, yamlDocument.RootNode)

	assert.Check(t, err)

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

	context := testHelpers.DefaultSettings()
	yamlDocument, _ := ParseFromContent(content, context, uri.File(""), protocol.Position{})

	diagnostics, err := handleYAMLErrors(err.Error(), content, yamlDocument.RootNode)

	assert.Check(t, err)

	expected := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 2, Character: 7},
			End:   protocol.Position{Line: 2, Character: 20},
		},
		Severity: protocol.DiagnosticSeverityError,
		Message:  protocol.String("yaml: unknown anchor 'unknownAnchor' referenced"),
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
			context := testHelpers.DefaultSettings()
			yamlDocument, _ := ParseFromContent([]byte(tc.yaml), context, uri.File(""), protocol.Position{})

			if tc.expectError {
				// For error cases, also run JSON schema validation
				validator := JSONSchemaValidator{
					Doc: yamlDocument,
				}

				err := validator.LoadJsonSchemaFromBytes(schema.EmbeddedSchemaJSON)
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

				assert.Check(t, len(diagnostics) != 0, "Expected validation errors but got none")
				if tc.expectErrorContains != "" {
					found := false
					for _, d := range diagnostics {
						if strings.Contains(strings.ToLower(diagnostic.MessageText(d)), strings.ToLower(tc.expectErrorContains)) {
							found = true
							break
						}
					}
					assert.Check(t, found, "Expected error message to contain '%s'", tc.expectErrorContains)
				}
			} else {
				// For non-error cases, just check that parsing succeeded
				assert.Assert(t, yamlDocument.Diagnostics != nil)
				diagnostics := *yamlDocument.Diagnostics
				assert.Check(t, cmp.Len(diagnostics, 0), "Expected no errors but got: %v", diagnostics)
			}
		})
	}
}

// yamlErrorDiagnostics is what handleYAMLErrors makes of the error yaml.v3
// reports for content.
func yamlErrorDiagnostics(t *testing.T, content string) []protocol.Diagnostic {
	t.Helper()

	var m map[string]any
	yamlErr := yaml.Unmarshal([]byte(content), &m)
	assert.Assert(t, yamlErr != nil, "content must not be valid YAML")

	yamlDocument, err := ParseFromContent([]byte(content), testHelpers.DefaultSettings(), uri.File(""), protocol.Position{})
	assert.NilError(t, err)
	t.Cleanup(yamlDocument.Close)

	diagnostics, err := handleYAMLErrors(yamlErr.Error(), []byte(content), yamlDocument.RootNode)
	assert.NilError(t, err)

	return diagnostics
}

func Test_HandleYAMLErrors_LineError(t *testing.T) {
	t.Run("is reported on the line it names", func(t *testing.T) {
		// yaml: line 2: could not find expected ':'
		diagnostics := yamlErrorDiagnostics(t, "a: 1\nb\nc: 2\n")

		assert.Assert(t, cmp.Len(diagnostics, 1))
		assert.Check(t, cmp.Equal(diagnostics[0].Range.Start.Line, uint32(1)))
		assert.Check(t, cmp.Contains(diagnostic.MessageText(diagnostics[0]), "Could not find expected ':'"))
	})

	t.Run("is reported on the last line when it names that one", func(t *testing.T) {
		// A key still being typed at the end of the file. This used to index
		// past the last line, and the panic took the server down.
		diagnostics := yamlErrorDiagnostics(t, "version: 2.1\nj")

		assert.Assert(t, cmp.Len(diagnostics, 1))
		assert.Check(t, cmp.Equal(diagnostics[0].Range.Start.Line, uint32(1)))
	})
}

func Test_HandleYamlError_UnknownAnchorWhereTheTreeHasNoNode(t *testing.T) {
	// The anchor's name is searched for in the whole text. Around the broken
	// tag, tree-sitter recovers with no node covering either place the name
	// is found, and the missing node used to be dereferenced. Found by
	// FuzzRequests.
	diagnostics := yamlErrorDiagnostics(t, "a: !0,0\nb: *0\n")

	assert.Check(t, cmp.Len(diagnostics, 0))
}

// schemaMessages is what validating content against the embedded schema says.
func schemaMessages(t *testing.T, content string) []string {
	t.Helper()

	yamlDocument, err := ParseFromContent([]byte(content), testHelpers.DefaultSettings(), uri.File(""), protocol.Position{})
	assert.NilError(t, err)
	t.Cleanup(yamlDocument.Close)

	validator := JSONSchemaValidator{Doc: yamlDocument}
	assert.NilError(t, validator.LoadEmbeddedJsonSchema())

	said := []string{}
	for _, d := range validator.ValidateWithJSONSchema(yamlDocument.RootNode, yamlDocument.Content) {
		said = append(said, diagnostic.MessageText(d))
	}

	return said
}

// A value that is only a reference is a string until the config is compiled,
// so the schema cannot say whether it is right.
func Test_ReferencesAreNotCheckedByTheSchema(t *testing.T) {
	job := func(body string) string {
		return `
version: 2.1
jobs:
  j:
    parameters:
      flag:
        type: boolean
        default: false
    docker:
      - image: cimg/base:stable
` + body + `
workflows:
  w:
    jobs:
      - j
`
	}

	t.Run("a boolean parameter where the schema wants a boolean", func(t *testing.T) {
		said := schemaMessages(t, job(`
    circleci_ip_ranges: << parameters.flag >>
    steps:
      - run: echo`))
		assert.Check(t, cmp.Len(said, 0))
	})

	t.Run("a parameter as the condition of a built-in step", func(t *testing.T) {
		said := schemaMessages(t, job(`
    steps:
      - checkout:
          when: << parameters.flag >>`))
		assert.Check(t, cmp.Len(said, 0))
	})

	t.Run("a pipeline value where the schema wants an integer", func(t *testing.T) {
		said := schemaMessages(t, job(`
    parallelism: << pipeline.number >>
    steps:
      - run: echo`))
		assert.Check(t, cmp.Len(said, 0))
	})

	t.Run("a wrong value beside a reference is still reported", func(t *testing.T) {
		said := schemaMessages(t, job(`
    circleci_ip_ranges: << parameters.flag >>
    parallelism: lots
    steps:
      - run: echo`))
		assert.Check(t, len(said) != 0, "a string parallelism must be rejected")
	})

	t.Run("a wrong literal value is still reported", func(t *testing.T) {
		said := schemaMessages(t, job(`
    circleci_ip_ranges: "yes"
    steps:
      - run: echo`))
		assert.Check(t, len(said) != 0, "a string circleci_ip_ranges must be rejected")
	})
}

func Test_OrbReferences(t *testing.T) {
	config := func(ref string) string {
		return `
version: 2.1
orbs:
  o: ` + ref + `
jobs:
  j:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo
workflows:
  w:
    jobs:
      - j
`
	}

	for _, ref := range []string{
		"circleci/node@7.1.0",
		"circleci/node@dev:alpha",
		"circleci/android@dev:exampleTag",
		"circleci/node@dev:feature/branch+1",
		"a/b@1",
	} {
		t.Run("accepts "+ref, func(t *testing.T) {
			assert.Check(t, cmp.Len(schemaMessages(t, config(ref)), 0))
		})
	}

	for _, ref := range []string{
		"circleci/node",
		"circleci/node@dev:",
		"Circleci/node@1.0.0",
		"circleci/node@latest",
	} {
		t.Run("rejects "+ref, func(t *testing.T) {
			assert.Check(t, len(schemaMessages(t, config(ref))) != 0, "%s must be rejected", ref)
		})
	}
}

func Test_Parallelism(t *testing.T) {
	config := func(parallelism string) string {
		return `
version: 2.1
jobs:
  j:
    docker:
      - image: cimg/base:stable
    parallelism: ` + parallelism + `
    steps:
      - run: echo
workflows:
  w:
    jobs:
      - j
`
	}

	for _, parallelism := range []string{"1", "4", "<< pipeline.number >>"} {
		t.Run("accepts "+parallelism, func(t *testing.T) {
			said := schemaMessages(t, config(parallelism))
			assert.Check(t, cmp.Len(said, 0))
		})
	}

	for _, parallelism := range []string{"0", "-1"} {
		t.Run("rejects "+parallelism, func(t *testing.T) {
			said := schemaMessages(t, config(parallelism))
			assert.Check(t, cmp.DeepEqual(said, []string{"Must be greater than or equal to 1"}))
		})
	}
}

func Test_WorkflowJobFilters(t *testing.T) {
	config := func(filters string) string {
		return `
version: 2.1
jobs:
  j:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo
workflows:
  w:
    jobs:
      - j:
          filters:
` + filters
	}

	t.Run("branches and tags", func(t *testing.T) {
		said := schemaMessages(t, config(`            branches:
              only: main
            tags:
              ignore: /.*/
`))
		assert.Check(t, cmp.Len(said, 0))
	})

	t.Run("any other key", func(t *testing.T) {
		said := schemaMessages(t, config(`            commits:
              only: main
`))
		assert.Check(t, cmp.DeepEqual(said, []string{"Additional property commits is not allowed"}))
	})
}
