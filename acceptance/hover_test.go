package acceptance

import (
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

// toolsOrbSource is an orb with a command and a job, each with a description.
const toolsOrbSource = `version: 2.1

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
}
