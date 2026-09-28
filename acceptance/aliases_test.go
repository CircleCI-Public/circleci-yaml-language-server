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

func TestOrbElementAliases(t *testing.T) {
	fake := linkedProjectFake(t)

	t.Run("executors", func(t *testing.T) {
		session := start(t, fake, executorAliasesConfig, testToken)
		diagnostics := session.open(t, executorAliasesConfig)

		assert.Check(t, cmp.DeepEqual(diagnostics, []string{}))
	})
}
