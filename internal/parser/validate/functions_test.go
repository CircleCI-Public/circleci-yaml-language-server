package validate

import (
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/codeaction"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

var anyOrder = cmpopts.SortSlices(func(a, b string) bool { return a < b })

// setupGoDescriptor is a trimmed copy of setup-go's function.yaml.
var setupGoDescriptor = map[string]any{
	"name":        "setup-go",
	"description": "Install a Go toolchain and link it onto PATH for later steps.",
	"version":     "v0.5.1-684fd5b",
	"flags": []any{
		map[string]any{"name": "version", "type": "string", "default": "stable", "description": "Go version spec."},
		map[string]any{"name": "version-file", "type": "string", "description": "Path to a go.mod file."},
		map[string]any{"name": "golangci-lint-cache", "type": "bool", "default": "false"},
	},
	"commands": map[string]any{
		"cache": map[string]any{
			"description": "Restore and save the Go caches.",
			"flags": []any{
				map[string]any{"name": "key", "type": "string"},
			},
		},
	},
}

// functionDiagnostics validates a config against a fake that publishes
// setup-go, and returns the messages of the given severity.
func functionDiagnostics(t *testing.T, severity protocol.DiagnosticSeverity, yamlContent string) []string {
	t.Helper()
	isolateOrbSources(t)

	fake := fakes.NewCircleCI(t)
	fake.SeedGoOrb()
	fake.AddFunction("fn-setup-go", "github.com/circleci-functions/setup-go", "Install a Go toolchain.",
		fakes.FunctionVersion{ID: "ver-setup-go-0-5-0", Version: "v0.5.0-aaaaaaa", Descriptor: map[string]any{"name": "setup-go"}},
		fakes.FunctionVersion{ID: "ver-setup-go-0-5-1", Version: "v0.5.1-684fd5b", Descriptor: setupGoDescriptor},
	)

	val := CreateValidateFromYAML(yamlContent)
	val.Context = testHelpers.SettingsForHost(fake.URL())
	val.Doc.Context = val.Context
	val.Cache.MachineOfferingsCache.Set(testMachineOfferings())
	val.Validate()

	messages := []string{}
	for _, d := range *val.Diagnostics {
		if d.Severity == severity {
			messages = append(messages, diagnostic.MessageText(d))
		}
	}
	return messages
}

func functionErrors(t *testing.T, yamlContent string) []string {
	t.Helper()
	return functionDiagnostics(t, protocol.DiagnosticSeverityError, yamlContent)
}

func functionWarnings(t *testing.T, yamlContent string) []string {
	t.Helper()
	return functionDiagnostics(t, protocol.DiagnosticSeverityWarning, yamlContent)
}

func TestFunctions(t *testing.T) {
	t.Run("a function step and its commands are accepted", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(functionErrors(t, `version: 2.1

functions:
  setup-go: github.com/circleci-functions/setup-go@v0.5.1-684fd5b

commands:
  setup:
    parameters:
      lint-cache:
        type: boolean
        default: false
    steps:
      - setup-go:
          with:
            version-file: go.mod
            golangci-lint-cache: << parameters.lint-cache >>

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - checkout
      - setup
      - setup-go
      - setup-go/cache:
          id: go-cache
      - when:
          condition: true
          steps:
            - setup-go:
                id: go

workflows:
  main:
    jobs:
      - build
`), []string{}, anyOrder))
	})

	t.Run("a step body takes only id and with", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(functionErrors(t, `version: 2.1

functions:
  setup-go: github.com/circleci-functions/setup-go@v0.5.1

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - setup-go:
          version: "1.24"
          id: Go
      - setup-go:
          with:
            versions:
              - "1.24"

workflows:
  main:
    jobs:
      - build
`), []string{
			"Unexpected key(s) in function step 'setup-go' in job 'build': version. Pass arguments under `with`.",
			`The function step 'setup-go' in job 'build' has an invalid id: "Go"`,
			"Argument(s) in the function step 'setup-go' in job 'build' must be a single value, not a list or map: versions",
		}, anyOrder))
	})

	t.Run("ids are unique within a job", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(functionErrors(t, `version: 2.1

functions:
  setup-go: github.com/circleci-functions/setup-go@v0.5.1

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - setup-go:
          id: go
      - setup-go/cache:
          id: go

workflows:
  main:
    jobs:
      - build
`), []string{
			"The job 'build' has more than one function step with id: go",
			"The job 'build' has more than one function step with id: go",
		}, anyOrder))
	})

	t.Run("a step names at most one command", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(functionErrors(t, `version: 2.1

functions:
  setup-go: github.com/circleci-functions/setup-go@v0.5.1

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - setup-go/cache/save

workflows:
  main:
    jobs:
      - build
`), []string{
			"The function step 'setup-go/cache/save' in job 'build' names more than one command. A step runs the function or one of its commands, and nothing deeper.",
		}, anyOrder))
	})

	t.Run("only the declarations that are used are checked", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(functionErrors(t, `version: 2.1

functions:
  no-version: github.com/circleci-functions/setup-go
  bad-path: setup-go@v0.1.0
  bad-version: github.com/circleci-functions/setup-go@0.1.0
  not-a-string:
    path: github.com/circleci-functions/setup-go
  unused: nonsense

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - no-version
      - bad-path
      - bad-version
      - not-a-string

workflows:
  main:
    jobs:
      - build
`), []string{
			"Function 'bad-path' has an invalid path: \"setup-go\". Expected host.tld/org/name, as in github.com/circleci-functions/setup-go@v0.1.0-abc1234",
			"Function 'bad-version' has an invalid version: \"0.1.0\". Expected a semver tag led by 'v', as in v0.1.0 or v0.1.0-abc1234",
			"Function 'no-version' is missing an '@version' suffix, which pins the release: github.com/circleci-functions/setup-go@v0.1.0-abc1234",
			"Function 'not-a-string' must be a string, for example github.com/circleci-functions/setup-go@v0.1.0-abc1234",
		}, anyOrder))
	})

	t.Run("an alias can't be an orb, a command or a built-in step", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(functionErrors(t, `version: 2.1

orbs:
  go: circleci/go@4.0.0

functions:
  go: github.com/circleci-functions/setup-go@v0.5.1
  greet: github.com/circleci-functions/greet@v0.5.1
  checkout: github.com/circleci-functions/checkout@v0.5.1

commands:
  greet:
    steps:
      - run: echo hello

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - greet

workflows:
  main:
    jobs:
      - build
`), []string{
			"Function name 'checkout' is also a built-in step. Rename one of them.",
			"Function name 'go' is also an orb. Rename one of them.",
			"Function name 'greet' is also a command. Rename one of them.",
		}, anyOrder))
	})

	t.Run("a functions block that isn't a map is left alone", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(functionErrors(t, `version: 2.1

functions: &anchors
  - &image cimg/base:current

jobs:
  build:
    docker:
      - image: *image
    steps:
      - checkout

workflows:
  main:
    jobs:
      - build
`), []string{}, anyOrder))
	})
}

func TestFunctionsInTheCatalog(t *testing.T) {
	config := func(declaration, steps string) string {
		return `version: 2.1

functions:
  setup-go: ` + declaration + `

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
` + steps + `
workflows:
  main:
    jobs:
      - build
`
	}

	t.Run("a step that matches the descriptor gets no warning", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(functionWarnings(t, config("github.com/circleci-functions/setup-go@v0.5.1-684fd5b", `      - setup-go:
          with:
            version: "1.24"
            golangci-lint-cache: true
      - setup-go/cache:
          with:
            key: v1
`)), []string{}))
	})

	t.Run("a function that isn't published", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(functionWarnings(t, config("github.com/circleci-functions/setup-rust@v0.1.0", "      - setup-go\n")), []string{
			"No function github.com/circleci-functions/setup-rust is published.",
		}))
	})

	t.Run("a version that isn't published", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(functionWarnings(t, config("github.com/circleci-functions/setup-go@v0.4.0", "      - setup-go\n")), []string{
			"Function github.com/circleci-functions/setup-go has no version v0.4.0. The latest is v0.5.1-684fd5b.",
		}))
	})

	t.Run("a command the version doesn't have", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(functionWarnings(t, config("github.com/circleci-functions/setup-go@v0.5.1-684fd5b", "      - setup-go/lint\n")), []string{
			"Function setup-go v0.5.1-684fd5b has no command lint.",
		}))
	})

	t.Run("flags the version doesn't take, or of the wrong type", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(functionWarnings(t, config("github.com/circleci-functions/setup-go@v0.5.1-684fd5b", `      - setup-go:
          with:
            versoin: "1.24"
            golangci-lint-cache: sometimes
      - setup-go/cache:
          with:
            version: "1.24"
`)), []string{
			"setup-go v0.5.1-684fd5b takes no flag versoin.",
			"Flag golangci-lint-cache of setup-go takes true or false.",
			"setup-go/cache v0.5.1-684fd5b takes no flag version.",
		}, anyOrder))
	})

	t.Run("an older patch gets a warning, and an update", func(t *testing.T) {
		val := validateWithSetupGo(t, config("github.com/circleci-functions/setup-go@v0.5.0-aaaaaaa", "      - setup-go\n"),
			fakes.FunctionVersion{ID: "ver-0-5-0", Version: "v0.5.0-aaaaaaa", Descriptor: map[string]any{"name": "setup-go"}},
			fakes.FunctionVersion{ID: "ver-0-5-1", Version: "v0.5.1-684fd5b", Descriptor: map[string]any{"name": "setup-go"}},
		)
		assert.Assert(t, cmp.Len(*val.Diagnostics, 1))
		got := (*val.Diagnostics)[0]
		assert.Check(t, cmp.Equal(got.Severity, protocol.DiagnosticSeverityWarning))
		assert.Check(t, cmp.Equal(diagnostic.MessageText(got),
			"A newer patch of github.com/circleci-functions/setup-go is published: v0.5.1-684fd5b."))
		assert.Check(t, cmp.DeepEqual(actionTitles(t, got), []string{"Update to v0.5.1-684fd5b"}))
	})

	t.Run("an older minor version gets information", func(t *testing.T) {
		val := validateWithSetupGo(t, config("github.com/circleci-functions/setup-go@v0.4.2-bbbbbbb", "      - setup-go\n"),
			fakes.FunctionVersion{ID: "ver-0-4-2", Version: "v0.4.2-bbbbbbb", Descriptor: map[string]any{"name": "setup-go"}},
			fakes.FunctionVersion{ID: "ver-0-5-1", Version: "v0.5.1-684fd5b", Descriptor: map[string]any{"name": "setup-go"}},
		)
		assert.Assert(t, cmp.Len(*val.Diagnostics, 1))
		got := (*val.Diagnostics)[0]
		assert.Check(t, cmp.Equal(got.Severity, protocol.DiagnosticSeverityInformation))
		assert.Check(t, cmp.Equal(diagnostic.MessageText(got),
			"A newer version of github.com/circleci-functions/setup-go is published: v0.5.1-684fd5b."))
	})

	t.Run("the latest version gets nothing", func(t *testing.T) {
		val := validateWithSetupGo(t, config("github.com/circleci-functions/setup-go@v0.5.1-684fd5b", "      - setup-go\n"),
			fakes.FunctionVersion{ID: "ver-0-5-1", Version: "v0.5.1-684fd5b", Descriptor: map[string]any{"name": "setup-go"}},
		)
		assert.Check(t, cmp.Len(*val.Diagnostics, 0))
	})

	t.Run("a host without the catalog gets no warning", func(t *testing.T) {
		isolateOrbSources(t)
		fake := fakes.NewCircleCI(t)
		fake.SetStatus("GET /api/v3/function/packages", 404)

		val := CreateValidateFromYAML(config("github.com/circleci-functions/setup-go@v0.5.1-684fd5b", "      - setup-go\n"))
		val.Context = testHelpers.SettingsForHost(fake.URL())
		val.Doc.Context = val.Context
		val.ValidateFunctions()

		assert.Check(t, cmp.Len(*val.Diagnostics, 0))
	})
}

// validateWithSetupGo validates the functions of a config against a catalog
// that publishes setup-go at the versions given.
func validateWithSetupGo(t *testing.T, config string, versions ...fakes.FunctionVersion) Validate {
	t.Helper()
	isolateOrbSources(t)
	fake := fakes.NewCircleCI(t)
	fake.AddFunction("fn-setup-go", "github.com/circleci-functions/setup-go", "Install a Go toolchain.", versions...)

	val := CreateValidateFromYAML(config)
	val.Context = testHelpers.SettingsForHost(fake.URL())
	val.Doc.Context = val.Context
	val.ValidateFunctions()
	return val
}

func actionTitles(t *testing.T, d protocol.Diagnostic) []string {
	t.Helper()
	actions, err := codeaction.FromData(d.Data)
	assert.NilError(t, err)

	titles := []string{}
	for _, action := range actions {
		titles = append(titles, action.Title)
	}
	return titles
}
