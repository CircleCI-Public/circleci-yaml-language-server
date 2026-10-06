package acceptance

import (
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/workspace"
)

// toolsOrbSource is an orb with a description, and a command and a job with
// one each.
const toolsOrbSource = `version: 2.1

description: Tools for acme.

jobs:
  test:
    description: Run the tests.
    machine:
      image: ubuntu-2404:current
    steps:
      - install

commands:
  install:
    description: Install the tools.
    parameters:
      version:
        type: string
        default: latest
    steps:
      - run: echo installing
`

// orbStepConfig runs the orb's command as a step, and the orb's job.
const orbStepConfig = `version: 2.1

orbs:
  tools: acme/tools@1.0.0

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
      - tools/test
`

func TestHover(t *testing.T) {
	fake := linkedProjectFake(t)
	fake.AddNamespace("ns-acme", "acme")
	fake.AddOrbPackage("orb-tools", "ns-acme", "acme", "tools", false, true)
	fake.AddOrbVersion("ver-tools", "orb-tools", "acme/tools", "1.0.0", toolsOrbSource, "")

	session := start(t, fake, orbStepConfig, testToken)
	session.open(t, orbStepConfig)

	markdownAt := func(t *testing.T, pos protocol.Position) string {
		t.Helper()
		hover, err := session.client.Hover(session.workspace.URI(), pos)
		assert.NilError(t, err)
		assert.Assert(t, hover != nil, "no hover")
		markup, ok := hover.Contents.(*protocol.MarkupContent)
		assert.Assert(t, ok, "hover contents are %T, not markup", hover.Contents)
		assert.Check(t, cmp.Equal(markup.Kind, protocol.MarkupKindMarkdown))
		return markup.Value
	}

	t.Run("a step shows the orb command it runs", func(t *testing.T) {
		got := markdownAt(t, position(10, 10))
		assert.Check(t, cmp.Equal(got,
			"**tools/install** command\n\nInstall the tools.\n\nParameters:\n\n- `version` (string, default `latest`)"))
	})

	t.Run("a workflow's job shows the orb job it runs", func(t *testing.T) {
		got := markdownAt(t, position(16, 10))
		assert.Check(t, cmp.Equal(got, "**tools/test** job\n\nRun the tests."))
	})

	t.Run("the orb's declaration shows the orb", func(t *testing.T) {
		got := markdownAt(t, position(3, 12))
		assert.Check(t, cmp.Equal(got,
			"**tools** orb `acme/tools@1.0.0`\n\nTools for acme.\n\nCommands: `install`\n\nJobs: `test`"))
	})
}

// functionStepConfig passes a flag to a declared function, on line 12.
const functionStepConfig = `version: 2.1

functions:
  setup-go: github.com/circleci-functions/setup-go@v0.5.3-4edeb2a

jobs:
  build:
    machine:
      image: ubuntu-2404:current
    steps:
      - setup-go:
          with:
            version: "1.22"

workflows:
  main:
    jobs:
      - build
`

func TestFunctionFlagHover(t *testing.T) {
	fake := linkedProjectFake(t)
	fake.AddFunction("fn-setup-go", "github.com/circleci-functions/setup-go", "Install a Go toolchain.",
		fakes.FunctionVersion{ID: "ver-setup-go", Version: "v0.5.3-4edeb2a", Descriptor: map[string]any{
			"name": "setup-go",
			"flags": []any{
				map[string]any{"name": "version", "type": "string", "default": "stable", "description": "Go version spec."},
			},
		}},
	)

	session := start(t, fake, functionStepConfig, testToken)
	session.open(t, functionStepConfig)

	hover, err := session.client.Hover(session.workspace.URI(), position(12, 14))
	assert.NilError(t, err)
	assert.Assert(t, hover != nil, "no hover")
	markup, ok := hover.Contents.(*protocol.MarkupContent)
	assert.Assert(t, ok, "hover contents are %T, not markup", hover.Contents)
	assert.Check(t, cmp.Equal(markup.Value, "**version** flag of **setup-go**\n\n(string, default `stable`): Go version spec."))
}

// resourceClassConfig has a job whose resource_class is on line 6.
const resourceClassConfig = `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:stable
    resource_class: large
    steps:
      - checkout

workflows:
  main:
    jobs:
      - build
`

func TestSchemaKeyHovers(t *testing.T) {
	resourceClass := position(6, 8)

	hoverAt := func(t *testing.T, options map[string]any) *protocol.Hover {
		t.Helper()
		fake := linkedProjectFake(t)
		session := startWithOptions(t, fake, workspace.New(t, resourceClassConfig), testToken, options)
		session.open(t, resourceClassConfig)

		hover, err := session.client.Hover(session.workspace.URI(), resourceClass)
		assert.NilError(t, err)
		assert.Assert(t, hover != nil)
		return hover
	}

	described := map[string]map[string]any{
		"a client that sends no options":       {},
		"the extension, when it asks for them": {"isCciExtension": true, "schemaHovers": true},
	}
	for name, options := range described {
		t.Run(name+" is shown the schema's description of a key", func(t *testing.T) {
			hover := hoverAt(t, options)
			markup, ok := hover.Contents.(*protocol.MarkupContent)
			assert.Assert(t, ok, "hover contents are %T, not markup", hover.Contents)
			assert.Check(t, cmp.Equal(markup.Kind, protocol.MarkupKindMarkdown))
			assert.Check(t, cmp.Contains(markup.Value, "Resource class for the job."))
		})
	}

	t.Run("an extension that shows them itself is not", func(t *testing.T) {
		hover := hoverAt(t, map[string]any{"isCciExtension": true})
		assert.Check(t, cmp.Nil(hover.Contents))
	})
}
