package validate

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/dockerhub"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

type ErrorTestCase struct {
	Name                   string
	YamlContent            string
	ExpectedDiagnosticLine uint32
}

func TestOrbValidation(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name:       "Local orb executor should give well located errors",
			OnlyErrors: true,
			YamlContent: `version: 2.1

orbs:
  localorb:
    executors:
      localexecutor:
        docker:
          - image: circleci/node`,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name: "Local mac orb executor should give well located diagnostics",
			YamlContent: `version: 2.1

orbs:
  localorb:
    executors:
      localmacexecutor:
        macos:
          xcode: 12.5`,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Warning(protocol.Range{
					Start: protocol.Position{Line: 3, Character: 2},
					End:   protocol.Position{Line: 7, Character: 21},
				}, "Orb is unused"),
				diagnostic.Warning(protocol.Range{
					Start: protocol.Position{Line: 7, Character: 10},
					End:   protocol.Position{Line: 7, Character: 21},
				},
					"Unknown Xcode version \"12.5\""),
			},
		},
		{
			Name:       "Local orb step should give well located errors",
			OnlyErrors: true,
			YamlContent: `version: 2.1

orbs:
  slack: circleci/slack@4.10.1
  localorb:
    commands:
      localcommand:
        steps:
          - run: echo "Hello world"
          - localorb/echo`,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 9, Character: 12},
					End:   protocol.Position{Line: 9, Character: 25},
				},
					"Cannot find declaration for step localorb/echo"),
			},
		},
		{
			Name:       "Local orb job should give well located errors",
			OnlyErrors: true,
			YamlContent: `version: 2.1

orbs:
  localorb:
    jobs:
      localjob:
        docker:
          - image: cimg/base:edge
        steps:
          - run: echo "Hello world"`,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			// This test is mainly here because checking an orb's executor would cause a crash
			Name: "Invalid remote orb",
			YamlContent: `version: 2.1

orbs:
  slack: circleci/toto@1.0.0

jobs:
  localjob:
    executor: slack/exec
    steps:
      - run: echo "Hello world"`,
			// We want an error on the orb and a warning on the executor
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 3, Character: 2},
					End:   protocol.Position{Line: 3, Character: 28},
				},
					"Orb circleci/toto does not exist or is private."),
				diagnostic.Warning(protocol.Range{
					Start: protocol.Position{Line: 7, Character: 4},
					End:   protocol.Position{Line: 7, Character: 24},
				},
					"Invalid orb or error trying to fetch it: could not find orb circleci/toto@1.0.0"),
				diagnostic.Warning(protocol.Range{
					Start: protocol.Position{Line: 6, Character: 2},
					End:   protocol.Position{Line: 6, Character: 10},
				},
					"Job is unused"),
			},
		},
		{
			Name: "Local orb with job",
			YamlContent: `version: 2.1,

orbs:
  localorb:
    jobs:
      localjob:
        docker:
          - image: cimg/base:2020.01
        steps:
          - run: echo "Hello world"

workflows:
  someworkflow:
    jobs:
      - localorb/localjob`,
			OnlyErrors: true,
		},
		{
			Name:       "Local orb with command",
			OnlyErrors: true,
			YamlContent: `version: 2.1

orbs:
  localorb:
    commands:
      localcommand:
        steps:
          - run: echo "Hello world"

jobs:
  somejob:
    docker:
      - image: cimg/base:2020.01
    steps:
      - localorb/localcommand

workflows:
  someworkflow:
    jobs:
      - somejob`,
		},
		{
			Name:       "Local orb with executor",
			OnlyErrors: true,
			YamlContent: `version: 2.1

orbs:
  localorb:
    executors:
      localexecutor:
        docker:
          - image: cimg/base:2020.01

jobs:
  somejob:
    executor: localorb/localexecutor
    steps:
      - run: echo "Hello world"

workflows:
  someworkflow:
    jobs:
      - somejob`,
		},
		// 		{
		// 			Name:       "Local orb with internal references",
		// 			OnlyErrors: true,
		// 			YamlContent: `version: 2.1

		// orbs:
		//   localorb:
		//     jobs:
		//       localjob:
		//         executor: localexecutor
		//         steps:
		//           - localcommand

		//     executors:
		//       localexecutor:
		//         docker:
		//           - image: cimg/base:2020.01

		//     commands:
		//       localcommand:
		//         steps:
		//           - run: echo "Hello world"

		// workflows:
		//   someworkflow:
		//     jobs:
		//       - localorb/localjob`,
		// 		},
		// 		{
		// 			Name:       "Local orb with special steps",
		// 			OnlyErrors: true,
		// 			YamlContent: `version: 2.1

		// orbs:
		//   localorb:
		//     jobs:
		//       localjob:
		//         docker:
		//           - image: cimg/base:2020.01
		//         steps:
		//           - checkout
		//           - special_save_cache
		//     commands:
		//       special_save_cache:
		//         steps:
		//           - save_cache

		// workflows:
		//   someworkflow:
		//     jobs:
		//       - localorb/localjob`,
		// 		},
		{
			Name:       "Local with strange positioned comment",
			OnlyErrors: true,
			YamlContent: `version: 2.1

orbs:
  localorb:
    jobs:
# some comment
      localjob:
        docker:
          - image: cimg/base:2020.01
        steps:
          - run: echo "Hello world"

workflows:
  someworkflow:
    jobs:
      - localorb/localjob`,
		},
	}

	CheckYamlErrors(t, testCases)
}

func TestOrbStepsUsedInParameters(t *testing.T) {
	content, err := os.ReadFile("testdata/orb_steps_used_in_params.yml")
	assert.Check(t, err)
	val := CreateValidateFromYAML(string(content))
	val.Validate()
	for _, diag := range *val.Diagnostics {
		if diag.Message == protocol.String("Orb is unused") {
			t.Errorf("Got orb is unused diagnostic")
		}
	}
}

func TestLocalOrbUsedPartsFalsePositive(t *testing.T) {
	fileURI := uri.File("some-uri")
	context := testHelpers.DefaultSettings()
	content, err := os.ReadFile("./testdata/orbs/local-orb-used-parts.yml")
	assert.Check(t, err)

	doc, err := parser.ParseFromContent(content, context, fileURI, protocol.Position{})
	assert.Check(t, err)

	val := Validate{
		APIs: ValidateAPIs{
			DockerHub: dockerhub.NewAPI(),
		},
		Diagnostics: &[]protocol.Diagnostic{},
		Cache:       cache.New(),
		Doc:         doc,
		Context:     context,
	}
	val.Validate()
	assert.Check(t, cmp.Len(*val.Diagnostics, 0))
}

func TestLocalOrbUnusedPartsFalseNegative(t *testing.T) {
	fileURI := uri.File("some-uri")
	context := testHelpers.DefaultSettings()
	content, err := os.ReadFile("./testdata/orbs/local-orb-unused-parts.yml")
	assert.Check(t, err)

	doc, err := parser.ParseFromContent(content, context, fileURI, protocol.Position{})
	assert.Check(t, err)

	val := Validate{
		APIs: ValidateAPIs{
			DockerHub: dockerhub.NewAPI(),
		},
		Diagnostics: &[]protocol.Diagnostic{},
		Cache:       cache.New(),
		Doc:         doc,
		Context:     context,
	}
	val.Validate()
	messages := getDiagnosticMessages(val.Diagnostics)
	assert.Check(t, cmp.DeepEqual(
		messages,
		[]string{"Orb is unused", "Job is unused", "Command is unused"},
		// The three warnings are independent, so their order is incidental.
		cmpopts.SortSlices(func(a, b string) bool { return a < b }),
	))
}

// A step can take other steps as a parameter, as aws-ecr/build_and_push_image
// takes its auth steps. Whatever is passed there is used.
func TestStepsPassedToAStepAreUsed(t *testing.T) {
	const wrapper = `
commands:
  wrap:
    parameters:
      inner:
        type: steps
    steps:
      - steps: << parameters.inner >>
  run-it:
    steps:
      - run: echo hi
`

	t.Run("an orb used only in a step's steps parameter", func(t *testing.T) {
		isolateOrbSources(t)

		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-acme", "acme")
		fake.AddOrbPackage("orb-in-param", "ns-acme", "acme", "in-param", false, true)
		fake.AddOrbVersion("ver-in-param-1", "orb-in-param", "acme/in-param", "1.0.0", orbSource, "")

		diagnostics := orbDiagnostics(t, fake, `version: 2.1
orbs:
  helper: acme/in-param@1.0.0
`+wrapper+`
jobs:
  j:
    docker:
      - image: cimg/base:stable
    steps:
      - wrap:
          inner:
            - helper/greet
`)
		assert.Check(t, !slices.Contains(getDiagnosticMessages(&diagnostics), "Orb is unused"), "diagnostics: %v", getDiagnosticMessages(&diagnostics))
	})

	t.Run("an orb used, with arguments, in a steps parameter nested in another", func(t *testing.T) {
		isolateOrbSources(t)

		fake := fakes.NewCircleCI(t)
		fake.AddNamespace("ns-acme", "acme")
		fake.AddOrbPackage("orb-nested", "ns-acme", "acme", "nested", false, true)
		fake.AddOrbVersion("ver-nested-1", "orb-nested", "acme/nested", "1.0.0", orbSource, "")

		diagnostics := orbDiagnostics(t, fake, `version: 2.1
orbs:
  helper: acme/nested@1.0.0
`+wrapper+`
jobs:
  j:
    docker:
      - image: cimg/base:stable
    steps:
      - wrap:
          inner:
            - wrap:
                inner:
                  - helper/greet:
                      name: with arguments
`)
		assert.Check(t, !slices.Contains(getDiagnosticMessages(&diagnostics), "Orb is unused"), "diagnostics: %v", getDiagnosticMessages(&diagnostics))
	})

	commandUnused := func(t *testing.T, yamlContent string) bool {
		t.Helper()

		val := CreateValidateFromYAML(yamlContent)
		val.Validate()

		return slices.Contains(getDiagnosticMessages(val.Diagnostics), "Command is unused")
	}

	t.Run("a command used only in a step's steps parameter", func(t *testing.T) {
		assert.Check(t, !commandUnused(t, `version: 2.1
`+wrapper+`
jobs:
  j:
    docker:
      - image: cimg/base:stable
    steps:
      - wrap:
          inner:
            - run-it
workflows:
  w:
    jobs:
      - j
`))
	})

	t.Run("a command used only in a job invocation's steps parameter", func(t *testing.T) {
		assert.Check(t, !commandUnused(t, `version: 2.1
`+wrapper+`
jobs:
  j:
    parameters:
      inner:
        type: steps
    docker:
      - image: cimg/base:stable
    steps:
      - wrap:
          inner: << parameters.inner >>
workflows:
  w:
    jobs:
      - j:
          inner:
            - run-it
`))
	})

	t.Run("a command that is not used anywhere is still reported", func(t *testing.T) {
		assert.Check(t, commandUnused(t, `version: 2.1
`+wrapper+`
jobs:
  j:
    docker:
      - image: cimg/base:stable
    steps:
      - wrap:
          inner:
            - run: echo
workflows:
  w:
    jobs:
      - j
`))
	})
}

// orb-tools/continue tests an orb by writing its source into the `{}` it is
// declared as, so what the orb declares is not known until then.
// https://circleci.com/docs/orbs/author/testing-orbs/
func TestInjectedOrbPlaceholder(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "Nothing referenced from an orb declared as {} is reported",
			YamlContent: `version: 2.1

orbs:
  my-orb: {}

jobs:
  integration-test:
    executor: my-orb/default
    steps:
      - my-orb/greet:
          to: world
      - my-orb/greet

workflows:
  test-deploy:
    jobs:
      - integration-test
      - my-orb/hello:
          name: hello-test
          to: world
`,
			OnlyErrors: true,
		},
	}

	CheckYamlErrors(t, testCases)
}

// Inside an inline orb, its own jobs and commands call its commands by their
// bare names.
func TestLocalOrbCommandUsedWithinTheOrb(t *testing.T) {
	unusedWarnings := func(t *testing.T, yamlContent string) []string {
		t.Helper()

		val := CreateValidateFromYAML(yamlContent)
		val.Validate()

		warnings := []string{}
		for _, message := range getDiagnosticMessages(val.Diagnostics) {
			if strings.HasSuffix(message, "is unused") {
				warnings = append(warnings, message)
			}
		}

		return warnings
	}

	t.Run("a command used by the orb's own job", func(t *testing.T) {
		assert.Check(t, cmp.Len(unusedWarnings(t, `version: 2.1
orbs:
  inline:
    commands:
      greet:
        steps:
          - run: echo hi
    jobs:
      hello:
        docker:
          - image: cimg/base:stable
        steps:
          - greet
workflows:
  w:
    jobs:
      - inline/hello
`), 0))
	})

	t.Run("a command used by another of the orb's commands", func(t *testing.T) {
		assert.Check(t, cmp.Len(unusedWarnings(t, `version: 2.1
orbs:
  inline:
    commands:
      greet:
        steps:
          - run: echo hi
      greet-twice:
        steps:
          - greet
          - greet
jobs:
  j:
    docker:
      - image: cimg/base:stable
    steps:
      - inline/greet-twice
workflows:
  w:
    jobs:
      - j
`), 0))
	})

	t.Run("a command the orb does not use is still reported", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(unusedWarnings(t, `version: 2.1
orbs:
  inline:
    commands:
      greet:
        steps:
          - run: echo hi
    jobs:
      hello:
        docker:
          - image: cimg/base:stable
        steps:
          - run: echo
workflows:
  w:
    jobs:
      - inline/hello
`), []string{"Command is unused"}))
	})
}

// The compiler fetches an orb referenced by URL from the prefixes an
// organization allows, which the server can't see.
func TestURLOrb(t *testing.T) {
	yamlContent := `version: 2.1

orbs:
  bp-go: https://raw.githubusercontent.com/circleci/backplane-cicd/refs/heads/main/orbs/go.yml

jobs:
  build:
    executor: bp-go/default
    steps:
      - bp-go/private-mod-init:
          private-modules: github.com/circleci/*

workflows:
  main:
    jobs:
      - build
      - bp-go/lint:
          name: lint
          context: org-global
`

	t.Run("nothing referenced from the orb is reported", func(t *testing.T) {
		CheckYamlErrors(t, []ValidateTestCase{{
			Name:        "no errors",
			YamlContent: yamlContent,
			OnlyErrors:  true,
		}})
	})

	t.Run("the orb gets one warning", func(t *testing.T) {
		val := CreateValidateFromYAML(yamlContent)
		val.ValidateOrbs()

		assert.Assert(t, cmp.Len(*val.Diagnostics, 1))
		diag := (*val.Diagnostics)[0]
		assert.Check(t, cmp.Equal(diag.Severity, protocol.DiagnosticSeverityWarning))
		assert.Check(t, cmp.Contains(diag.Message, "not fetched"))
		assert.Check(t, cmp.Equal(diag.Range.Start, protocol.Position{Line: 3, Character: 9}))
	})
}
