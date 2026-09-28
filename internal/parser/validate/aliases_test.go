package validate

import (
	"testing"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
)

func span(line, start, end uint32) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{Line: line, Character: start},
		End:   protocol.Position{Line: line, Character: end},
	}
}

func TestExecutorAliases(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "an alias of a local executor passes its arguments on",
			YamlContent: `version: 2.1
executors:
  alias-exec: real-exec
  real-exec:
    parameters:
      tag:
        type: string
    machine:
      image: ubuntu-2404:<< parameters.tag >>
jobs:
  build:
    executor:
      name: alias-exec
      nope: current
    steps:
      - run: echo hello
workflows:
  workflow:
    jobs:
      - build
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 11, Character: 4},
					End:   protocol.Position{Line: 13, Character: 19},
				}, "Parameter tag is required for real-exec"),
				diagnostic.Error(span(13, 6, 19), "Parameter nope is not defined for real-exec"),
			},
		},
		{
			Name: "an alias of an executor the config doesn't define",
			YamlContent: `version: 2.1
executors:
  my-exec: small
jobs:
  build:
    executor: my-exec
    steps:
      - run: echo hello
workflows:
  workflow:
    jobs:
      - build
`,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(2, 11, 16),
					"Executor alias 'my-exec' refers to 'small', which is not a known local executor"),
			},
		},
		{
			Name: "an alias of an executor an orb doesn't have",
			YamlContent: `version: 2.1
orbs:
  electric:
    executors:
      default:
        machine:
          image: ubuntu-2404:current
executors:
  bad-exec: electric/nonexistent
jobs:
  build:
    executor: bad-exec
    steps:
      - run: echo hello
workflows:
  workflow:
    jobs:
      - build
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(8, 12, 32), "Cannot find executor nonexistent in orb electric"),
			},
		},
		{
			Name: "an alias of an orb the config doesn't declare",
			YamlContent: `version: 2.1
executors:
  bad-exec: nope/default
jobs:
  build:
    executor: bad-exec
    steps:
      - run: echo hello
workflows:
  workflow:
    jobs:
      - build
`,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(2, 12, 24),
					"Unable to determine target for executor invocation nope/default (renamed from local executor bad-exec)"),
			},
		},
		{
			Name: "an alias with more than one slash",
			YamlContent: `version: 2.1
executors:
  bad-exec: a/b/c
jobs:
  build:
    executor: bad-exec
    steps:
      - run: echo hello
workflows:
  workflow:
    jobs:
      - build
`,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(2, 12, 17), malformedAliasMessage),
			},
		},
		{
			Name: "an alias nothing uses is only a warning",
			YamlContent: `version: 2.1
executors:
  unused-exec: small
jobs:
  build:
    machine:
      image: ubuntu-2404:current
    steps:
      - run: echo hello
workflows:
  workflow:
    jobs:
      - build
`,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Warning(span(2, 2, 13), "Executor is unused"),
				diagnostic.Warning(span(2, 15, 20),
					"Executor alias 'unused-exec' refers to 'small', which is not a known local executor"),
			},
		},
		{
			Name: "an executor used only through an alias is used",
			YamlContent: `version: 2.1
executors:
  alias-exec: real-exec
  real-exec:
    machine:
      image: ubuntu-2404:current
jobs:
  build:
    executor: alias-exec
    steps:
      - run: echo hello
workflows:
  workflow:
    jobs:
      - build
`,
		},
		{
			Name: "an inline orb's job using the orb's own alias",
			YamlContent: `version: 2.1
orbs:
  myorb:
    executors:
      real-exec:
        machine:
          image: ubuntu-2404:current
      string-exec: real-exec
    jobs:
      do-thing:
        executor: string-exec
        steps:
          - run: echo hello
workflows:
  workflow:
    jobs:
      - myorb/do-thing
`,
		},
	}

	CheckYamlErrors(t, testCases)
}
