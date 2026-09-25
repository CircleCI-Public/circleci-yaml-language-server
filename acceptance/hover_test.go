package acceptance

import (
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

// toolsOrbSource is an orb with a command that has a description and a
// parameter.
const toolsOrbSource = `version: 2.1

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

// orbStepConfig runs the orb's command as a step.
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
`

func TestHover(t *testing.T) {
	t.Run("a step shows the orb command it runs", func(t *testing.T) {
		fake := linkedProjectFake(t)
		fake.AddNamespace("ns-acme", "acme")
		fake.AddOrbPackage("orb-tools", "ns-acme", "acme", "tools", false, true)
		fake.AddOrbVersion("ver-tools", "orb-tools", "acme/tools", "1.0.0", toolsOrbSource, "")

		session := start(t, fake, orbStepConfig, testToken)
		session.open(t, orbStepConfig)

		hover, err := session.client.Hover(session.workspace.URI(), position(10, 10))
		assert.NilError(t, err)
		assert.Assert(t, hover != nil, "no hover")
		markup, ok := hover.Contents.(*protocol.MarkupContent)
		assert.Assert(t, ok, "hover contents are %T, not markup", hover.Contents)
		assert.Check(t, cmp.Equal(markup.Kind, protocol.MarkupKindMarkdown))
		assert.Check(t, cmp.Equal(markup.Value,
			"**tools/install** command\n\nInstall the tools.\n\nParameters:\n\n- `version` (string, default `latest`)"))
	})
}
