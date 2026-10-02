package parser

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
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
	yamlDocument, _ := ParseFromContent(t.Context(), content, context, uri.File(""), protocol.Position{})

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
	yamlDocument, _ := ParseFromContent(t.Context(), content, context, uri.File(""), protocol.Position{})

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
			yamlDocument, _ := ParseFromContent(t.Context(), []byte(tc.yaml), context, uri.File(""), protocol.Position{})

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

				schemaDiagnostics, yamlDiagnostics := validator.ValidateWithJSONSchema(yamlDocument.RootNode, yamlDocument.Content)
				diagnostics := append(schemaDiagnostics, yamlDiagnostics...)

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

	yamlDocument, err := ParseFromContent(t.Context(), []byte(content), testHelpers.DefaultSettings(), uri.File(""), protocol.Position{})
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
func schemaDiagnostics(t *testing.T, content string) []protocol.Diagnostic {
	t.Helper()

	yamlDocument, err := ParseFromContent(t.Context(), []byte(content), testHelpers.DefaultSettings(), uri.File(""), protocol.Position{})
	assert.NilError(t, err)
	t.Cleanup(yamlDocument.Close)

	validator := JSONSchemaValidator{Doc: yamlDocument}
	assert.NilError(t, validator.LoadEmbeddedJsonSchema())

	schemaDiagnostics, yamlDiagnostics := validator.ValidateWithJSONSchema(yamlDocument.RootNode, yamlDocument.Content)
	return append(schemaDiagnostics, yamlDiagnostics...)
}

func schemaMessages(t *testing.T, content string) []string {
	t.Helper()

	said := []string{}
	for _, d := range schemaDiagnostics(t, content) {
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
		assert.Check(t, cmp.DeepEqual(said, []string{"`commits` isn't allowed here."}))
	})
}

func Test_DockerLayerCaching(t *testing.T) {
	config := func(value string) string {
		return `
version: 2.1
executors:
  vm:
    parameters:
      dlc:
        type: boolean
        default: false
    machine:
      image: ubuntu-2204:current
      docker_layer_caching: ` + value + `
jobs:
  j:
    executor: vm
    steps:
      - run: echo
workflows:
  w:
    jobs:
      - j
`
	}

	for _, value := range []string{"true", "yes", "On", "<< parameters.dlc >>"} {
		t.Run("accepts "+value, func(t *testing.T) {
			said := schemaMessages(t, config(value))
			assert.Check(t, cmp.Len(said, 0))
		})
	}

	for _, value := range []string{`"yes"`, "1"} {
		t.Run("rejects "+value, func(t *testing.T) {
			said := schemaMessages(t, config(value))
			assert.Check(t, len(said) != 0, "%s must be rejected", value)
		})
	}
}

func Test_TeardownSteps(t *testing.T) {
	config := func(teardown string) string {
		return `
version: 2.1
jobs:
  j:
    docker:
      - image: cimg/base:stable
    steps:
      - save_cache: {key: "", paths: [a-path]}
      - run:
          command: make build
          teardown:
            - ` + teardown + `
workflows:
  w:
    jobs:
      - j
`
	}

	t.Run("a save_cache key can't be empty, in a teardown or not", func(t *testing.T) {
		said := schemaMessages(t, config(`save_cache: {key: "", paths: [a-path]}`))
		assert.Check(t, cmp.DeepEqual(said, []string{
			"String length must be greater than or equal to 1",
			"String length must be greater than or equal to 1",
		}))
	})

	t.Run("a teardown run can be rerun", func(t *testing.T) {
		said := schemaMessages(t, config(`run: {command: ./save.sh, max_auto_reruns: 2, auto_rerun_delay: 30s}`))
		assert.Check(t, cmp.DeepEqual(said, []string{"String length must be greater than or equal to 1"}))
	})

	t.Run("a teardown run's rerun delay needs a number of reruns", func(t *testing.T) {
		said := schemaMessages(t, config(`run: {command: ./save.sh, auto_rerun_delay: 30s}`))
		assert.Check(t, cmp.Contains(said, "max_auto_reruns is required"))
	})
}

func Test_DeploySteps(t *testing.T) {
	config := func(step string) string {
		return `
version: 2.1
jobs:
  j:
    docker:
      - image: cimg/base:stable
    steps:
      - ` + step + `
workflows:
  w:
    jobs:
      - j
`
	}

	accepted := []string{
		`deploy: ./deploy.sh`,
		`deploy: {command: ./deploy.sh}`,
		`deploy: {name: Deploy, command: ./deploy.sh, shell: /bin/bash, working_directory: ~/app}`,
		`deploy: {command: ./deploy.sh, background: true, no_output_timeout: 20m, when: on_success}`,
		`deploy: {command: ./deploy.sh, environment: {STAGE: prod}}`,
		`deploy: {command: ./deploy.sh, max_auto_reruns: 2, auto_rerun_delay: 30s}`,
	}
	for _, step := range accepted {
		t.Run("accepts "+step, func(t *testing.T) {
			said := schemaMessages(t, config(step))
			assert.Check(t, cmp.Len(said, 0))
		})
	}

	t.Run("checks its options as run's", func(t *testing.T) {
		said := schemaMessages(t, config(`deploy: {command: ./deploy.sh, max_auto_reruns: 9}`))
		assert.Check(t, cmp.DeepEqual(said, []string{"Must be less than or equal to 5"}))
	})

	t.Run("suggests the run option a misspelt one was meant to be", func(t *testing.T) {
		said := schemaMessages(t, config(`deploy: {command: ./deploy.sh, shel: /bin/bash}`))
		assert.Check(t, cmp.DeepEqual(said, []string{"deploy has no shel option, so this is ignored. Did you mean `shell`?"}))
	})
}

func Test_InvalidNames(t *testing.T) {
	diags := schemaDiagnostics(t, `
version: 2.1
executors:
  myExec:
    docker:
      - image: cimg/base:stable
commands:
  say-hello:
    steps:
      - run: echo hi
  otherCommand:
    steps:
      - run: ls
`)

	type reported struct {
		Line    uint32
		Message string
	}
	var got []reported
	for _, d := range diags {
		got = append(got, reported{d.Range.Start.Line, diagnostic.MessageText(d)})
	}

	assert.Check(t, cmp.DeepEqual(got, []reported{
		{3, `"myExec" isn't a valid executor name: it must start with a lowercase letter, and have only lowercase letters, digits, "_" and "-".`},
		{10, `"otherCommand" isn't a valid command name: it must start with a lowercase letter, and have only lowercase letters, digits, "_" and "-".`},
	}, cmpopts.SortSlices(func(a, b reported) bool { return a.Line < b.Line })))
}

func Test_JobWithoutExecutor(t *testing.T) {
	config := func(executor string) string {
		return `
version: 2.1
executors:
  e:
    docker:
      - image: cimg/base:stable
jobs:
  deploy:` + executor + `
    steps:
      - checkout
`
	}

	t.Run("has none", func(t *testing.T) {
		diags := schemaDiagnostics(t, config(""))
		assert.Assert(t, cmp.Len(diags, 1))
		assert.Check(t, cmp.Equal(diagnostic.MessageText(diags[0]),
			"A job needs an executor: give it one of `docker`, `machine`, `macos` or `executor`."))
		assert.Check(t, cmp.DeepEqual(diags[0].Range, protocol.Range{
			Start: protocol.Position{Line: 7, Character: 2},
			End:   protocol.Position{Line: 7, Character: 8},
		}))
	})

	executors := map[string]string{
		"docker":   "\n    docker:\n      - image: cimg/base:stable",
		"machine":  "\n    machine:\n      image: ubuntu-2404:current",
		"macos":    "\n    macos:\n      xcode: 16.0.0",
		"executor": "\n    executor: e",
	}
	for name, executor := range executors {
		t.Run("has "+name, func(t *testing.T) {
			said := schemaMessages(t, config(executor))
			assert.Check(t, cmp.Len(said, 0))
		})
	}
}

func Test_HandleYAMLErrors_CollectionKey(t *testing.T) {
	content := []byte(`workflows:
  main:
    jobs:
      - build:
          context: {{ .ContextName }}
      - test:
          matrix: { [a, b]: 1 }
`)

	var file interface{}
	yamlErr := yaml.Unmarshal(content, &file)
	assert.Assert(t, yamlErr != nil)

	context := testHelpers.DefaultSettings()
	yamlDocument, _ := ParseFromContent(t.Context(), content, context, uri.File(""), protocol.Position{})

	diagnostics, err := handleYAMLErrors(yamlErr.Error(), content, yamlDocument.RootNode)
	assert.NilError(t, err)

	expected := []protocol.Diagnostic{
		diagnostic.Error(protocol.Range{
			Start: protocol.Position{Line: 4, Character: 20},
			End:   protocol.Position{Line: 4, Character: 36},
		}, "A map can't be used as a key; quote it if it's meant as text"),
		diagnostic.Error(protocol.Range{
			Start: protocol.Position{Line: 6, Character: 20},
			End:   protocol.Position{Line: 6, Character: 26},
		}, "A list can't be used as a key; quote it if it's meant as text"),
	}
	assert.Check(t, cmp.Len(diagnostics, len(expected)))
	expect.DiagnosticList(t, diagnostics).To.IncludeAll(expected)
}

func Test_KeysTheSchemaDoesNotAllow(t *testing.T) {
	job := func(body string) string {
		return `version: 2.1
jobs:
  build:
    docker:
      - image: cimg/base:current
` + body + `
workflows:
  w:
    jobs: [build]
`
	}

	testCases := []struct {
		name string
		body string
		want string
		// The text the diagnostic covers.
		covers string
	}{
		{
			name:   "a misspelt job key",
			body:   "    resouce_class: large\n    steps: [checkout]",
			want:   "`resouce_class` isn't allowed here. Did you mean `resource_class`?",
			covers: "resouce_class",
		},
		{
			name:   "a misspelt Docker image key",
			body:   "        entrypont: [sh]\n    steps: [checkout]",
			want:   "`entrypont` isn't allowed here. Did you mean `entrypoint`?",
			covers: "entrypont",
		},
		{
			name:   "a misspelt parameter key",
			body:   "    parameters:\n      p:\n        type: string\n        defualt: x\n    steps: [checkout]",
			want:   "`defualt` isn't allowed here. Did you mean `default`?",
			covers: "defualt",
		},
		{
			name:   "a key like none allowed",
			body:   "    banana: 1\n    steps: [checkout]",
			want:   "`banana` isn't allowed here.",
			covers: "banana",
		},
		{
			name:   "a misspelt option of an open step, which is ignored",
			body:   "    steps:\n      - run:\n          command: echo\n          shel: bash",
			want:   "run has no shel option, so this is ignored. Did you mean `shell`?",
			covers: "shel",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			content := job(tc.body)
			diagnostics := schemaDiagnostics(t, content)
			assert.Assert(t, cmp.Len(diagnostics, 1))

			message := diagnostic.MessageText(diagnostics[0])
			assert.Check(t, cmp.Equal(message, tc.want))
			rng := diagnostics[0].Range
			assert.Assert(t, cmp.Equal(rng.Start.Line, rng.End.Line), "covers one line")
			line := strings.Split(content, "\n")[rng.Start.Line]
			covered := line[rng.Start.Character:rng.End.Character]
			assert.Check(t, cmp.Equal(covered, tc.covers))
		})
	}
}

// The compiler reads a plain yes, no, on or off as a boolean, as YAML 1.1 does.
func Test_YAML11Booleans(t *testing.T) {
	job := func(body string) string {
		return `version: 2.1
jobs:
  build:
    docker:
      - image: cimg/base:current
` + body + `
workflows:
  w:
    jobs: [build]
`
	}

	t.Run("rejected where text is wanted", func(t *testing.T) {
		for _, value := range []string{"yes", "No", "on", "OFF"} {
			content := job("    working_directory: " + value + "\n    steps: [checkout]")
			diagnostics := schemaDiagnostics(t, content)
			assert.Assert(t, cmp.Len(diagnostics, 1), value)

			message := diagnostic.MessageText(diagnostics[0])
			assert.Check(t, cmp.Equal(message, "`"+value+"` is read as a boolean; quote it if it's meant as text."))
			rng := diagnostics[0].Range
			line := strings.Split(content, "\n")[rng.Start.Line]
			covered := line[rng.Start.Character:rng.End.Character]
			assert.Check(t, cmp.Contains(covered, value))
		}
	})

	t.Run("text when quoted", func(t *testing.T) {
		for _, value := range []string{`"yes"`, `'off'`} {
			said := schemaMessages(t, job("    working_directory: "+value+"\n    steps: [checkout]"))
			assert.Check(t, cmp.Len(said, 0), value)
		}
	})

	t.Run("text when not a YAML 1.1 boolean", func(t *testing.T) {
		for _, value := range []string{"y", "n", "yess", "onward"} {
			said := schemaMessages(t, job("    working_directory: "+value+"\n    steps: [checkout]"))
			assert.Check(t, cmp.Len(said, 0), value)
		}
	})

	t.Run("accepted where a boolean may be", func(t *testing.T) {
		said := schemaMessages(t, job(`    environment:
      DEBUG: yes
    steps:
      - run:
          command: sleep 10
          background: on`))
		assert.Check(t, cmp.Len(said, 0))
	})
}
