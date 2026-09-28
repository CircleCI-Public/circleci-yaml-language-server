package acceptance

import (
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

// executorAliasesConfig names its executors through aliases: one of an inline
// orb's executor, one of an executor it defines, and one inside the orb.
const executorAliasesConfig = `version: 2.1

orbs:
  electric:
    executors:
      default:
        machine:
          image: ubuntu-2404:current
      real-exec:
        machine:
          image: ubuntu-2404:current
      string-exec: real-exec
    jobs:
      do-thing:
        executor: string-exec
        steps:
          - run: echo hello

executors:
  my-exec: electric/default
  alias-exec: real-exec
  real-exec:
    machine:
      image: ubuntu-2404:current

jobs:
  via-orb:
    executor: my-exec
    steps:
      - run: echo hello
  via-local:
    executor: alias-exec
    steps:
      - run: echo hello

workflows:
  main:
    jobs:
      - via-orb
      - via-local
      - electric/do-thing
`

// commandAliasesConfig renames an orb's command, and calls it both from a
// job's steps and from a workflow's post-steps. The orb's own job still finds
// the orb's command rather than the config's command of the same name.
const commandAliasesConfig = `version: 2.1

orbs:
  orb:
    commands:
      c:
        steps:
          - run: echo orb command
    jobs:
      build:
        machine:
          image: ubuntu-2404:current
        steps:
          - c

commands:
  c:
    steps:
      - run: echo local command
  renamed-c: orb/c

jobs:
  use-renamed-c:
    machine:
      image: ubuntu-2404:current
    steps:
      - renamed-c

workflows:
  workflow:
    jobs:
      - orb/build:
          pre-steps:
            - c
          post-steps:
            - renamed-c
      - use-renamed-c
`

// jobAliasesConfig renames an orb's job, which still uses the orb's own
// executor and command, and runs it with an argument.
const jobAliasesConfig = `version: 2.1

orbs:
  orb:
    executors:
      e:
        machine:
          image: ubuntu-2404:current
    commands:
      c:
        steps:
          - run: echo orb command
    jobs:
      build:
        parameters:
          greeting:
            type: string
            default: hello
        executor: e
        steps:
          - c
          - run: echo << parameters.greeting >>

jobs:
  build: orb/build

workflows:
  workflow:
    jobs:
      - build:
          greeting: hi
`

// implicitWorkflowConfig has no workflows, so the compiler runs the job
// named build, which here is an alias.
const implicitWorkflowConfig = `version: 2.1

orbs:
  orb:
    jobs:
      build:
        machine:
          image: ubuntu-2404:current
        steps:
          - checkout

jobs:
  build: orb/build
`

// registryAliasesConfig renames a published orb's command and job.
const registryAliasesConfig = `version: 2.1

orbs:
  tools: acme/tools@1.0.0

commands:
  install: tools/install

jobs:
  test: tools/test
  build:
    machine:
      image: ubuntu-2404:current
    steps:
      - install

workflows:
  main:
    jobs:
      - build
      - test
`

func TestOrbElementAliases(t *testing.T) {
	fake := linkedProjectFake(t)

	t.Run("executors", func(t *testing.T) {
		session := start(t, fake, executorAliasesConfig, testToken)
		diagnostics := session.open(t, executorAliasesConfig)

		assert.Check(t, cmp.DeepEqual(diagnostics, []string{}))
	})

	t.Run("commands", func(t *testing.T) {
		session := start(t, fake, commandAliasesConfig, testToken)
		diagnostics := session.open(t, commandAliasesConfig)

		assert.Check(t, cmp.DeepEqual(diagnostics, []string{}))
	})

	t.Run("jobs", func(t *testing.T) {
		session := start(t, fake, jobAliasesConfig, testToken)
		diagnostics := session.open(t, jobAliasesConfig)

		assert.Check(t, cmp.DeepEqual(diagnostics, []string{}))
	})

	t.Run("a published orb's command and job", func(t *testing.T) {
		fake.AddNamespace("ns-acme", "acme")
		fake.AddOrbPackage("orb-tools", "ns-acme", "acme", "tools", false, true)
		fake.AddOrbVersion("ver-tools", "orb-tools", "acme/tools", "1.0.0", toolsOrbSource, "")

		session := start(t, fake, registryAliasesConfig, testToken)
		diagnostics := session.open(t, registryAliasesConfig)

		t.Run("have nothing wrong with them", func(t *testing.T) {
			assert.Check(t, cmp.DeepEqual(diagnostics, []string{}))
		})

		t.Run("a step shows the command its alias names", func(t *testing.T) {
			hover, err := session.client.Hover(session.workspace.URI(), position(14, 10))
			assert.NilError(t, err)
			assert.Assert(t, hover != nil, "no hover")
			markup, ok := hover.Contents.(*protocol.MarkupContent)
			assert.Assert(t, ok, "hover contents are %T, not markup", hover.Contents)
			assert.Check(t, cmp.Equal(markup.Value,
				"**install** command\n\nInstall the tools.\n\nParameters:\n\n- `version` (string, default `latest`)"))
		})
	})

	t.Run("a job run by the implicit workflow", func(t *testing.T) {
		session := start(t, fake, implicitWorkflowConfig, testToken)
		diagnostics := session.open(t, implicitWorkflowConfig)

		assert.Check(t, cmp.DeepEqual(diagnostics, []string{}))
	})
}
