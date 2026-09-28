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

func TestCommandAliases(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "an alias of an orb's command takes its parameters",
			YamlContent: `version: 2.1
orbs:
  orb:
    commands:
      c:
        parameters:
          greeting:
            type: string
        steps:
          - run: echo << parameters.greeting >>
commands:
  renamed-c: orb/c
jobs:
  build:
    machine:
      image: ubuntu-2404:current
    steps:
      - renamed-c:
          greeting: hello
      - renamed-c:
          nope: hello
workflows:
  workflow:
    jobs:
      - build
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(19, 8, 17), "Parameter greeting is required for renamed-c"),
				diagnostic.Error(span(20, 10, 21), "Parameter nope is not defined for renamed-c"),
			},
		},
		{
			Name: "an alias of an orb the config doesn't declare",
			YamlContent: `version: 2.1
commands:
  renamed-c: nope/c
jobs:
  build:
    machine:
      image: ubuntu-2404:current
    steps:
      - renamed-c:
          anything: at all
workflows:
  workflow:
    jobs:
      - build
`,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(2, 13, 19),
					"Unable to determine target for step invocation nope/c (renamed from local command renamed-c)"),
			},
		},
		{
			Name: "an alias of a command an orb doesn't have",
			YamlContent: `version: 2.1
orbs:
  orb:
    jobs:
      c:
        machine:
          image: ubuntu-2404:current
        steps:
          - run: echo hello
commands:
  renamed-c: orb/c
jobs:
  build:
    machine:
      image: ubuntu-2404:current
    steps:
      - renamed-c
workflows:
  workflow:
    jobs:
      - build
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(10, 13, 18), "orb/c is a job, not a command: a job can't be run as a step"),
			},
		},
		{
			Name: "a malformed alias that is invoked",
			YamlContent: `version: 2.1
commands:
  checkout: 45m
jobs:
  build:
    machine:
      image: ubuntu-2404:current
    steps:
      - checkout
workflows:
  workflow:
    jobs:
      - build
`,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Warning(span(2, 2, 10),
					"Command 'checkout' in commands.checkout shadows built-in CircleCI command 'checkout'"),
				diagnostic.Error(span(2, 12, 15), malformedAliasMessage),
			},
		},
		{
			Name: "a malformed alias nothing invokes",
			YamlContent: `version: 2.1
commands:
  no_output_timeout: 45m
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
				diagnostic.Warning(span(2, 2, 19), "Command is unused"),
				diagnostic.Warning(span(2, 21, 24), "`45m` is not a valid orb element alias, "+
					"so this entry is ignored unless it is invoked. An alias must be a single "+
					"`orb-alias/element-name` reference."),
			},
		},
		{
			Name: "a malformed alias named run, which nothing can invoke",
			YamlContent: `version: 2.1
commands:
  run: 45m
jobs:
  build:
    machine:
      image: ubuntu-2404:current
    steps:
      - checkout
workflows:
  workflow:
    jobs:
      - build
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(2, 7, 10), malformedAliasMessage),
			},
		},
		{
			Name: "an orb's command used only through an alias is used",
			YamlContent: `version: 2.1
orbs:
  orb:
    commands:
      c:
        steps:
          - run: echo hello
commands:
  renamed-c: orb/c
jobs:
  build:
    machine:
      image: ubuntu-2404:current
    steps:
      - renamed-c
workflows:
  workflow:
    jobs:
      - build:
          post-steps:
            - renamed-c
`,
		},
		{
			Name: "an inline orb's alias of an orb it declares",
			YamlContent: `version: 2.1
orbs:
  outer:
    orbs:
      inner:
        commands:
          c:
            steps:
              - run: echo inner
    commands:
      renamed-c: inner/c
    jobs:
      use-renamed-c:
        machine:
          image: ubuntu-2404:current
        steps:
          - renamed-c
workflows:
  workflow:
    jobs:
      - outer/use-renamed-c
`,
		},
		{
			Name: "an inline orb's malformed alias that it invokes",
			YamlContent: `version: 2.1
orbs:
  outer:
    commands:
      renamed-c: 45m
    jobs:
      use-renamed-c:
        machine:
          image: ubuntu-2404:current
        steps:
          - renamed-c
workflows:
  workflow:
    jobs:
      - outer/use-renamed-c
`,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(4, 17, 20), malformedAliasMessage),
			},
		},
	}

	CheckYamlErrors(t, testCases)
}
