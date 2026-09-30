package acceptance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

// These configs are shaped like ones in our own orgs that the server used to
// report errors in, which the compiler accepts.

// acmeToolsOrbSource is acme/tools: an executor, a command and a job, each
// taking a parameter.
const acmeToolsOrbSource = `version: 2.1

executors:
  default:
    machine:
      image: ubuntu-2404:current

commands:
  install:
    parameters:
      version:
        type: string
        default: latest
    steps:
      - run: echo installing << parameters.version >>

jobs:
  lint:
    parameters:
      strict:
        type: boolean
        default: false
    executor: default
    steps:
      - install
      - run: echo linting
`

// urlOrbConfig uses an executor, a command and a job of an orb given by a URL
// the server can't fetch from, as it can't for a private one the compiler
// fetches with its organization's credentials.
const urlOrbConfig = `version: 2.1

orbs:
  tools: https://127.0.0.1:1/orbs/tools.yml

jobs:
  build:
    executor: tools/default
    steps:
      - tools/install:
          version: 1.2.3

workflows:
  main:
    jobs:
      - build
      - tools/lint:
          name: lint
          strict: true
`

// inlineOrbExecutorConfig runs a job on an inline orb's executor, named
// directly and as an executor parameter's default.
const inlineOrbExecutorConfig = `version: 2.1

orbs:
  local:
    executors:
      linux:
        machine:
          image: ubuntu-2404:current

jobs:
  build:
    executor: local/linux
    steps:
      - run: echo building
  package:
    parameters:
      exec:
        type: executor
        default: local/linux
    executor: << parameters.exec >>
    steps:
      - run: echo packaging

workflows:
  main:
    jobs:
      - build
      - package
`

// nestedOrbsConfig has an inline orb that declares a published orb and an
// inline orb of its own, and calls a command of each.
const nestedOrbsConfig = `version: 2.1

orbs:
  outer:
    orbs:
      tools: acme/tools@1.0.0
      inner:
        commands:
          hi:
            steps:
              - run: echo hi
    jobs:
      use-them:
        machine:
          image: ubuntu-2404:current
        steps:
          - tools/install
          - inner/hi

workflows:
  main:
    jobs:
      - outer/use-them
`

// inlineOrbOwnCommandsConfig has an inline orb whose commands are called only
// by its own job and its own commands, by their bare names.
const inlineOrbOwnCommandsConfig = `version: 2.1

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
      hello:
        machine:
          image: ubuntu-2404:current
        steps:
          - greet-twice

workflows:
  main:
    jobs:
      - inline/hello
`

// stepsArgumentOrbConfig uses a published orb only inside the steps a command
// is given as an argument.
const stepsArgumentOrbConfig = `version: 2.1

orbs:
  tools: acme/tools@1.0.0

commands:
  with-setup:
    parameters:
      setup:
        type: steps
    steps:
      - steps: << parameters.setup >>
      - run: echo done

jobs:
  build:
    machine:
      image: ubuntu-2404:current
    steps:
      - with-setup:
          setup:
            - tools/install:
                version: 1.2.3

workflows:
  main:
    jobs:
      - build
`

// devLabelConfig uses a development version of an orb whose label has
// capitals in it.
const devLabelConfig = `version: 2.1

orbs:
  tools: acme/tools@dev:exampleTag

jobs:
  build:
    machine:
      image: ubuntu-2404:current
    steps:
      - tools/install

workflows:
  main:
    jobs:
      - build
`

// yttTemplateConfig is a ytt template, which isn't config until ytt fills in
// its values.
const yttTemplateConfig = `#@ load("@ytt:data", "data")
version: 2.1

jobs:
  lint:
    machine:
      image: ubuntu-2404:current
    parameters:
      partition:
        type: enum
        enum: #@ data.values.partitions
    steps:
      - run: echo linting

workflows:
  main:
    jobs:
      - lint
`

// testSuites is a Smarter Testing definition, which lives in .circleci beside
// the config.
const testSuites = `name: ci tests
discover: go list ./...
run: gotestsum -- << test.atoms >>
outputs:
  junit: test-reports/tests.xml
---
name: windows
run: gotestsum -- ./...
`

// gitHubWorkflow is a GitHub Actions workflow. It has jobs and steps, as
// pipeline config does, so checked as config it would get errors.
const gitHubWorkflow = `name: CI
on: push
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
`

func TestNoFalseErrorsFromOrbs(t *testing.T) {
	fake := linkedProjectFake(t)
	// acme/tools, released as 1.0.0 and as the development version
	// dev:exampleTag.
	fake.AddNamespace("ns-acme", "acme")
	fake.AddOrbPackage("orb-tools", "ns-acme", "acme", "tools", false, true)
	fake.AddOrbVersion("ver-tools-1", "orb-tools", "acme/tools", "1.0.0", acmeToolsOrbSource, "")
	fake.AddOrbVersion("ver-tools-dev", "orb-tools", "acme/tools", "dev:exampleTag", acmeToolsOrbSource, "")

	t.Run("an orb at a URL that can't be fetched gets one warning", func(t *testing.T) {
		session := start(t, fake, urlOrbConfig, testToken)
		diagnostics := session.open(t, urlOrbConfig)

		assert.Assert(t, cmp.Len(diagnostics, 1), "diagnostics: %v", diagnostics)
		// The reason is the operating system's.
		assert.Check(t, cmp.Regexp(`^Could not fetch this orb \(.+\), so nothing used from it is checked\.$`, diagnostics[0]))
	})

	for _, tc := range []struct {
		name   string
		config string
	}{
		{"an inline orb's executor is found in the orb", inlineOrbExecutorConfig},
		{"an inline orb calls the orbs it declares for itself", nestedOrbsConfig},
		{"an inline orb's commands called only by the orb are used", inlineOrbOwnCommandsConfig},
		{"an orb used only in a steps argument is used", stepsArgumentOrbConfig},
		{"a development label may have capitals", devLabelConfig},
	} {
		t.Run(tc.name, func(t *testing.T) {
			session := start(t, fake, tc.config, testToken)
			diagnostics := session.open(t, tc.config)

			assert.Check(t, cmp.DeepEqual(diagnostics, []string{}))
		})
	}
}

func TestFilesThatAreNotConfig(t *testing.T) {
	fake := linkedProjectFake(t)

	t.Run("a ytt template isn't checked", func(t *testing.T) {
		session := start(t, fake, yttTemplateConfig, testToken)
		diagnostics := session.open(t, yttTemplateConfig)

		assert.Check(t, cmp.DeepEqual(diagnostics, []string{}))
	})

	t.Run("a Smarter Testing definition beside the config isn't checked", func(t *testing.T) {
		session := start(t, fake, validConfig, testToken)
		path := filepath.Join(session.workspace.Root, ".circleci", "test-suites.yml")
		err := os.WriteFile(path, []byte(testSuites), 0o600)
		assert.NilError(t, err)

		err = session.client.DidOpen(uri.File(path), testSuites)
		assert.NilError(t, err)
		diagnostics, err := session.client.WaitForDiagnostics(uri.File(path))
		assert.NilError(t, err)

		assert.Check(t, cmp.DeepEqual(messages(diagnostics), []string{}))
	})

	t.Run("YAML outside .circleci is left alone", func(t *testing.T) {
		session := start(t, fake, validConfig, testToken)
		workflow := uri.File(filepath.Join(session.workspace.Root, ".github", "workflows", "ci.yml"))

		t.Run("open a GitHub workflow, and then the config", func(t *testing.T) {
			err := session.client.DidOpen(workflow, gitHubWorkflow)
			assert.NilError(t, err)

			// The workflow was opened first, so checking it would have been
			// done by the time the config's diagnostics arrive.
			assert.Check(t, cmp.DeepEqual(session.open(t, validConfig), []string{}))
		})

		t.Run("nothing is published for the workflow", func(t *testing.T) {
			_, err := session.client.WaitForDiagnosticsWithin(workflow, 0)
			assert.Check(t, cmp.ErrorContains(err, "no diagnostics published"))
		})

		t.Run("requests about the workflow are answered with null", func(t *testing.T) {
			document := protocol.TextDocumentIdentifier{URI: workflow}
			at := protocol.TextDocumentPositionParams{TextDocument: document, Position: position(2, 2)}
			for method, params := range map[string]any{
				protocol.MethodTextDocumentHover:              protocol.HoverParams{TextDocumentPositionParams: at},
				protocol.MethodTextDocumentDefinition:         protocol.DefinitionParams{TextDocumentPositionParams: at},
				protocol.MethodTextDocumentReferences:         protocol.ReferenceParams{TextDocumentPositionParams: at},
				protocol.MethodTextDocumentCompletion:         protocol.CompletionParams{TextDocumentPositionParams: at},
				protocol.MethodTextDocumentDocumentSymbol:     protocol.DocumentSymbolParams{TextDocument: document},
				protocol.MethodTextDocumentSemanticTokensFull: protocol.SemanticTokensParams{TextDocument: document},
			} {
				var result json.RawMessage
				err := session.client.Call(method, params, &result)
				assert.Check(t, err, method)
				assert.Check(t, cmp.Equal(string(result), "null"), method)
			}
		})
	})
}
