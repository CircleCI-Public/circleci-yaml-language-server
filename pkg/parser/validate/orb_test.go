package validate

import (
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

type ErrorTestCase struct {
	Name                   string
	YamlContent            string
	ExpectedDiagnosticLine uint32
}

func TestOrbValidation(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name:       "Local orb executor should give well located errors",
			OnlyErrors: true,
			YamlContent: `version: 2.1

orbs:
  localorb:
    executors:
      localexecutor:
        docker:
          - image: circleci/node`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 7, Character: 12},
					End:   protocol.Position{Line: 7, Character: 32},
				},
					"Missing image tag"),
			},
		},
		{
			Name:       "Local mac orb executor should give well located errors",
			OnlyErrors: true,
			YamlContent: `version: 2.1

orbs:
  localorb:
    executors:
      localmacexecutor:
        macos:
          xcode: 12.5`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 7, Character: 10},
					End:   protocol.Position{Line: 7, Character: 21},
				},
					"Invalid Xcode version 12.5"),
			},
		},
		{
			Name:       "Local orb step should give well located errors",
			OnlyErrors: true,
			YamlContent: `version: 2.1

orbs:
  slack: circleci/slack@4.10.1
  localorb:
    commands:
      localcommand:
        steps:
          - echo "Hello world"`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 8, Character: 12},
					End:   protocol.Position{Line: 8, Character: 30},
				},
					"Cannot find declaration for step localorb/echo \"Hello world\""),
			},
		},
		{
			Name:       "Local orb job should give well located errors",
			OnlyErrors: true,
			YamlContent: `version: 2.1

orbs:
  localorb:
    jobs:
      localjob:
        docker:
          - image: cimg/base
        steps:
          - run: echo "Hello world"`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 7, Character: 12},
					End:   protocol.Position{Line: 7, Character: 28},
				},
					"Missing image tag",
				),
			},
		},
		{
			// This test is mainly here because checking an orb's executor would cause a crash
			Name:       "Invalid remote orb",
			OnlyErrors: true,
			YamlContent: `version: 2.1

orbs:
  slack: circleci/toto@1.0.0

jobs:
  localjob:
    executor: slack/exec
  steps:
    - run: echo "Hello world"`,
			// We want an error on the orb and a warning on the executor
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 3, Character: 2},
					End:   protocol.Position{Line: 3, Character: 28},
				},
					"Cannot find remote orb circleci/toto@1.0.0"),
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 7, Character: 4},
					End:   protocol.Position{Line: 7, Character: 24},
				},
					"Cannot find executor exec in orb slack"),
			},
		},
		{
			Name: "Local orb with job",
			YamlContent: `version: 2.1

orbs:
  localorb:
    jobs:
      localjob:
        docker:
          - image: cimg/base:2020.01
        steps:
          - run: echo "Hello world"

workflows:
  someworkflow:
    jobs:
      - localorb/localjob`,
		},
		{
			Name: "Local orb with command",
			YamlContent: `version: 2.1

orbs:
  localorb:
    commands:
      localcommand:
        steps:
          - run: echo "Hello world"

jobs:
  somejob:
    docker:
      - image: cimg/base:2020.01
    steps:
      - localorb/localcommand

workflows:
  someworkflow:
    jobs:
      - somejob`,
		},
		{
			Name: "Local orb with executor",
			YamlContent: `version: 2.1

orbs:
  localorb:
    executors:
      localexecutor:
        docker:
          - image: cimg/base:2020.01

jobs:
  somejob:
    executor: localorb/localexecutor
    steps:
      - run: echo "Hello world"

workflows:
  someworkflow:
    jobs:
      - somejob`,
		},
		{
			Name: "Local orb with internal references",
			YamlContent: `version: 2.1

orbs:
  localorb:
    jobs:
      localjob:
        executor: localexecutor
        steps:
          - localcommand

    executors:
      localexecutor:
        docker:
          - image: cimg/base:2020.01

    commands:
      localcommand:
        steps:
          - run: echo "Hello world"

workflows:
  someworkflow:
    jobs:
      - localorb/localjob`,
		},
		{
			Name: "Local orb with special steps",
			YamlContent: `version: 2.1

orbs:
  localorb:
    jobs:
      localjob:
        docker:
          - image: cimg/base:2020.01
        steps:
          - checkout
          - special_save_cache
    commands:
      special_save_cache:
        steps:
          - save_cache

workflows:
  someworkflow:
    jobs:
      - localorb/localjob`,
		},
		{
			Name: "Local with strange positioned comment",
			YamlContent: `version: 2.1

orbs:
  localorb:
    jobs:
# some comment
      localjob:
        docker:
          - image: cimg/base:2020.01
        steps:
          - run: echo "Hello world"

workflows:
  someworkflow:
    jobs:
      - localorb/localjob`,
		},
	}

	CheckYamlErrors(t, testCases)
}
