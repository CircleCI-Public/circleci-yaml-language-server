package acceptance

import (
	"testing"

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
}
