package validate

import (
	"fmt"
	"slices"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func validateYAML(t *testing.T, yamlData string) *[]protocol.Diagnostic {
	t.Helper()
	ctx := testHelpers.DefaultSettings()
	doc, err := parser.ParseFromContent(
		[]byte(yamlData),
		ctx,
		uri.URI(""),
		protocol.Position{},
	)
	assert.Check(t, err, "invalid YAML data")

	val := Validate{
		APIs:        ValidateAPIs{DockerHubMock{}},
		Context:     ctx,
		Doc:         doc,
		Diagnostics: &[]protocol.Diagnostic{},
		Cache:       testHelpers.DefaultCache(),
	}
	val.Validate()
	return val.Diagnostics
}

func diagnosticMessages(diags *[]protocol.Diagnostic) []string {
	msgs := make([]string, len(*diags))
	for i, d := range *diags {
		msgs[i] = diagnostic.MessageText(d)
	}
	return msgs
}

func TestJobUsedInJobGroup_GroupUsedInWorkflow(t *testing.T) {
	yamlData := `version: 2.1

jobs:
  deploy:
    docker:
      - image: cimg/base:2024.01
    steps:
      - run: echo deploy
  release:
    docker:
      - image: cimg/base:2024.01
    steps:
      - run: echo release

job-groups:
  deploy-group:
    jobs:
      - deploy
      - release:
          requires:
            - deploy

workflows:
  main:
    jobs:
      - deploy-group`

	diags := validateYAML(t, yamlData)
	msgs := diagnosticMessages(diags)

	for _, msg := range msgs {
		if msg == "Job is unused" {
			t.Fatalf("expected no 'Job is unused' diagnostic but got one.\nAll diagnostics: %v", msgs)
		}
	}
}

func TestJobUsedInJobGroup_GroupNotUsedInWorkflow(t *testing.T) {
	yamlData := `version: 2.1

jobs:
  deploy:
    docker:
      - image: cimg/base:2024.01
    steps:
      - run: echo deploy
  release:
    docker:
      - image: cimg/base:2024.01
    steps:
      - run: echo release
  build:
    docker:
      - image: cimg/base:2024.01
    steps:
      - run: echo build

job-groups:
  deploy-group:
    jobs:
      - deploy
      - release:
          requires:
            - deploy

workflows:
  main:
    jobs:
      - build`

	diags := validateYAML(t, yamlData)
	msgs := diagnosticMessages(diags)

	expectedMsgs := []string{
		`Job "deploy" is used in job group "deploy-group", but that group is never invoked in a workflow`,
		`Job "release" is used in job group "deploy-group", but that group is never invoked in a workflow`,
	}
	for _, expected := range expectedMsgs {
		found := false
		for _, msg := range msgs {
			if msg == expected {
				found = true
				break
			}
		}
		assert.Check(t, found, "expected diagnostic: %q\nAll diagnostics: %v", expected, msgs)
	}
}

func TestJobUnusedInWorkflow(t *testing.T) {
	yamlData := `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:2024.01
    steps:
      - run: echo build
  unused-job:
    docker:
      - image: cimg/base:2024.01
    steps:
      - run: echo unused

workflows:
  main:
    jobs:
      - build`

	diags := validateYAML(t, yamlData)
	msgs := diagnosticMessages(diags)

	hasUnused := false
	for _, msg := range msgs {
		if msg == "Job is unused" {
			hasUnused = true
		}
	}
	assert.Check(t, hasUnused, "expected 'Job is unused' diagnostic for unused-job.\nAll diagnostics: %v", msgs)
}

func TestBuildJobRunsWithoutWorkflows(t *testing.T) {
	diags := validateYAML(t, `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:2024.01
    steps:
      - run: echo build
  unused-job:
    docker:
      - image: cimg/base:2024.01
    steps:
      - run: echo unused
`)

	var unused []uint32
	for _, diag := range *diags {
		if diagnostic.MessageText(diag) == "Job is unused" {
			unused = append(unused, diag.Range.Start.Line)
		}
	}
	assert.Check(t, cmp.DeepEqual(unused, []uint32{8}), "only unused-job is unused")
}

func TestJobGroupUnusedInWorkflow(t *testing.T) {
	yamlData := `version: 2.1

jobs:
  deploy:
    docker:
      - image: cimg/base:2024.01
    steps:
      - run: echo deploy

job-groups:
  deploy-group:
    jobs:
      - deploy

workflows:
  main:
    jobs:
      - deploy`

	diags := validateYAML(t, yamlData)
	msgs := diagnosticMessages(diags)

	hasUnusedGroup := false
	for _, msg := range msgs {
		if msg == "Job group is unused" {
			hasUnusedGroup = true
		}
	}
	assert.Check(t, hasUnusedGroup, "expected 'Job group is unused' diagnostic.\nAll diagnostics: %v", msgs)
}

func TestExecutorParam(t *testing.T) {
	testCases := []struct {
		label        string
		yamlData     string
		expectedDiag protocol.Diagnostic
	}{
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
			ctx := &session.Settings{
				Api: circleci.Config{
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
			assert.Check(t, err, "invalid YAML data")
			assert.Check(t, cmp.Contains(doc.Jobs, "test"))

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

func TestExecutorParamWithoutDefault(t *testing.T) {
	diags := validateYAML(t, `version: 2.1

executors:
  linux:
    docker:
      - image: cimg/base:2024.01

jobs:
  build:
    parameters:
      executor:
        type: executor
    executor: << parameters.executor >>
    steps:
      - checkout

workflows:
  given:
    jobs:
      - build:
          executor: linux
  missing:
    jobs:
      - build
`)

	var got []protocol.Diagnostic
	for _, diag := range *diags {
		if diag.Severity <= protocol.DiagnosticSeverityWarning {
			got = append(got, diag)
		}
	}
	assert.Assert(t, cmp.Len(got, 1), "diagnostics: %v", diagnosticMessages(diags))
	assert.Check(t, cmp.Equal(diagnostic.MessageText(got[0]), "Parameter executor is required for build"))
	assert.Check(t, cmp.Equal(got[0].Severity, protocol.DiagnosticSeverityError))
	assert.Check(t, cmp.Equal(got[0].Range.Start.Line, uint32(23)))
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
			label: "machine resource_class",
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
				Severity: protocol.DiagnosticSeverityWarning,
			},
		},
		{
			label: "macos resource_class",
			yamlData: `jobs:
  test:
    macos:
      xcode: 26.5.0
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
			ctx := &session.Settings{
				Api: circleci.Config{
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
			assert.Check(t, err, "invalid YAML data")
			assert.Check(t, cmp.Contains(doc.Jobs, "test"))

			val := Validate{
				APIs:        ValidateAPIs{DockerHubMock{}},
				Context:     ctx,
				Doc:         doc,
				Diagnostics: &[]protocol.Diagnostic{},
				Cache:       cache.New(),
			}
			val.Cache.MachineOfferingsCache.Set(testHelpers.MachineOfferings())
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
      caches: 31d
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
			ctx := &session.Settings{
				Api: circleci.Config{
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
			assert.Check(t, err, "invalid YAML data")
			assert.Check(t, cmp.Contains(doc.Jobs, "test"))

			val := Validate{
				APIs:        ValidateAPIs{DockerHubMock{}},
				Context:     ctx,
				Doc:         doc,
				Diagnostics: &[]protocol.Diagnostic{},
				Cache:       cache.New(),
			}
			val.Cache.MachineOfferingsCache.Set(testHelpers.MachineOfferings())
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
				Message:  protocol.String("If no `type:` key is specified, the job will default to `type: build`."),
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
				Message:  protocol.String("Invalid job type 'bad-type'. Allowed types: approval, build, no-op, release, lock, unlock"),
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
				Message:  protocol.String("Steps only exist in `build` jobs. Steps here will be ignored."),
			},
		},
	}

	for _, testCase := range testCases {
		t.Run("validate job type: "+testCase.label, func(t *testing.T) {
			ctx := &session.Settings{
				Api: circleci.Config{
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
			assert.Check(t, err, "invalid YAML data")

			val := Validate{
				APIs:        ValidateAPIs{DockerHubMock{}},
				Context:     ctx,
				Doc:         doc,
				Diagnostics: &[]protocol.Diagnostic{},
				Cache:       cache.New(),
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

func TestReservedJobParameterNames(t *testing.T) {
	diags := validateYAML(t, `version: 2.1

jobs:
  build:
    parameters:
      context:
        type: string
        default: ""
      pre-steps:
        type: steps
        default: []
      words:
        type: string
        default: ""
    docker:
      - image: cimg/base:2024.01
    steps:
      - run: echo << parameters.context >> << parameters.words >>
      - steps: << parameters.pre-steps >>

workflows:
  main:
    jobs:
      - build
`)

	const reserved = `"name", "pre-steps", "post-steps", "filters", "requires", "context", "type", ` +
		`"override-with", and "upstream" are reserved parameter names in build jobs`
	var lines []uint32
	for _, diag := range *diags {
		if diagnostic.MessageText(diag) == reserved {
			lines = append(lines, diag.Range.Start.Line)
		}
	}
	assert.Check(t, cmp.DeepEqual(lines, []uint32{8, 5}), "pre-steps, then context")
}

func TestTypelessExecutor(t *testing.T) {
	diags := validateYAML(t, `version: 2.1

executors:
  big:
    resource_class: large

jobs:
  typed-here:
    executor: big
    docker:
      - image: cimg/base:2024.01
    steps:
      - checkout
  typed-nowhere:
    executor: big
    steps:
      - checkout

workflows:
  main:
    jobs:
      - typed-here
      - typed-nowhere
`)

	var got []string
	for _, diag := range *diags {
		if diag.Severity == protocol.DiagnosticSeverityError {
			got = append(got, fmt.Sprintf("%d: %s", diag.Range.Start.Line, diagnostic.MessageText(diag)))
		}
	}
	assert.Check(t, cmp.DeepEqual(got, []string{
		`14: Executor big is missing a required key: "docker", "machine", or "macos"`,
	}))
}

func TestStoreTestResultsHint(t *testing.T) {
	const hint = "You may want to add the `store_test_results` step to visualize the test results in CircleCI"

	hinted := func(t *testing.T, yamlData string) bool {
		t.Helper()

		messages := diagnosticMessages(validateYAML(t, yamlData))
		return slices.Contains(messages, hint)
	}

	job := func(steps string) string {
		return `
jobs:
  test:
    docker:
      - image: cimg/base:stable
    steps:
` + steps + `
workflows:
  main:
    jobs:
      - test
`
	}

	const commands = `version: 2.1

commands:
  store:
    steps:
      - store_test_results:
          path: reports
  store-indirectly:
    steps:
      - store
  wrap:
    parameters:
      steps:
        type: steps
    steps:
      - steps: << parameters.steps >>
  loop:
    steps:
      - loop
`

	cases := []struct {
		name   string
		config string
		hinted bool
	}{
		{
			name:   "no results stored",
			config: commands + job("      - run: go test ./..."),
			hinted: true,
		},
		{
			name: "results stored by the job",
			config: commands + job(`      - run: go test ./...
      - store_test_results:
          path: reports`),
		},
		{
			name:   "results stored through a command",
			config: commands + job("      - store"),
		},
		{
			name:   "results stored through a command that calls another",
			config: commands + job("      - store-indirectly"),
		},
		{
			name: "results stored inside when",
			config: commands + job(`      - when:
          condition: true
          steps:
            - store`),
		},
		{
			name: "results stored through a steps argument",
			config: commands + job(`      - wrap:
          steps:
            - store`),
		},
		{
			name: "a steps argument that stores nothing",
			config: commands + job(`      - wrap:
          steps:
            - run: go test ./...`),
			hinted: true,
		},
		{
			name:   "a command that calls itself",
			config: commands + job("      - loop"),
			hinted: true,
		},
		{
			name: "results stored through an inline orb's command",
			config: commands + `
orbs:
  reports:
    commands:
      upload:
        steps:
          - save
      save:
        steps:
          - store_test_results:
              path: reports
` + job("      - reports/upload"),
		},
		{
			name: "an inline orb's command that stores nothing",
			config: commands + `
orbs:
  reports:
    commands:
      upload:
        steps:
          - run: echo nothing
` + job("      - reports/upload"),
			hinted: true,
		},
		{
			name: "a command from an orb that can't be fetched",
			config: `version: 2.1

parameters:
  orb:
    type: string
    default: acme/reports@1.0.0

orbs:
  reports: << pipeline.parameters.orb >>
` + job("      - reports/upload"),
		},
		{
			name: "steps passed to the job",
			config: `version: 2.1

jobs:
  test:
    parameters:
      steps:
        type: steps
    docker:
      - image: cimg/base:stable
    steps:
      - steps: << parameters.steps >>

workflows:
  main:
    jobs:
      - test:
          steps:
            - run: go test ./...
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := hinted(t, tc.config)
			assert.Check(t, cmp.Equal(got, tc.hinted))
		})
	}
}

func TestJobWithoutAMappingIsReportedAtItsName(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		// The lines, zero-based, of the jobs expected to be reported.
		lines []uint32
	}{
		{
			name: "steps indented as a job",
			yaml: `version: 2.1
jobs:
  build:
    docker:
      - image: cimg/base:current
    steps: [checkout]
  steps:
    - checkout
workflows:
  main:
    jobs:
      - build
`,
			lines: []uint32{6},
		},
		{
			name: "two jobs without a value",
			yaml: `version: 2.1
jobs:
  deploy:
  test:
`,
			lines: []uint32{2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := validateYAML(t, tt.yaml)

			var lines []uint32
			for _, d := range *diags {
				assert.Check(t, !position.IsDefaultRange(d.Range), "%q is at the start of the file", diagnostic.MessageText(d))
				if diagnostic.MessageText(d) == "Job is unused" {
					lines = append(lines, d.Range.Start.Line)
				}
			}
			slices.Sort(lines)
			assert.Check(t, cmp.DeepEqual(lines, tt.lines))
		})
	}
}
