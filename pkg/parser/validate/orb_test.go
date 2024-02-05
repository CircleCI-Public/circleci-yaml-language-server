package validate

import (
	"os"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/dockerhub"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/testHelpers"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
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
			Diagnostics: []protocol.Diagnostic{},
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
          - run: echo "Hello world"
          - localorb/echo`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 9, Character: 12},
					End:   protocol.Position{Line: 9, Character: 25},
				},
					"Cannot find declaration for step localorb/echo"),
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
					- image: cimg/base:edge
        steps:
          - run: echo "Hello world"`,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			// This test is mainly here because checking an orb's executor would cause a crash
			Name: "Invalid remote orb",
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
					"Orb circleci/toto does not exist or is private."),
				utils.CreateWarningDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 7, Character: 4},
					End:   protocol.Position{Line: 7, Character: 24},
				},
					"Invalid orb or error trying to fetch it: could not find orb circleci/toto@1.0.0"),
				utils.CreateWarningDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 6, Character: 2},
					End:   protocol.Position{Line: 6, Character: 10},
				},
					"Job is unused"),
			},
		},
		{
			Name: "Local orb with job",
			YamlContent: `version: 2.1,

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
			OnlyErrors: true,
		},
		{
			Name:       "Local orb with command",
			OnlyErrors: true,
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
			Name:       "Local orb with executor",
			OnlyErrors: true,
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
		// 		{
		// 			Name:       "Local orb with internal references",
		// 			OnlyErrors: true,
		// 			YamlContent: `version: 2.1

		// orbs:
		//   localorb:
		//     jobs:
		//       localjob:
		//         executor: localexecutor
		//         steps:
		//           - localcommand

		//     executors:
		//       localexecutor:
		//         docker:
		//           - image: cimg/base:2020.01

		//     commands:
		//       localcommand:
		//         steps:
		//           - run: echo "Hello world"

		// workflows:
		//   someworkflow:
		//     jobs:
		//       - localorb/localjob`,
		// 		},
		// 		{
		// 			Name:       "Local orb with special steps",
		// 			OnlyErrors: true,
		// 			YamlContent: `version: 2.1

		// orbs:
		//   localorb:
		//     jobs:
		//       localjob:
		//         docker:
		//           - image: cimg/base:2020.01
		//         steps:
		//           - checkout
		//           - special_save_cache
		//     commands:
		//       special_save_cache:
		//         steps:
		//           - save_cache

		// workflows:
		//   someworkflow:
		//     jobs:
		//       - localorb/localjob`,
		// 		},
		{
			Name:       "Local with strange positioned comment",
			OnlyErrors: true,
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

func TestOrbStepsUsedInParameters(t *testing.T) {
	content, err := os.ReadFile("testdata/orb_steps_used_in_params.yml")
	assert.NoError(t, err)
	val := CreateValidateFromYAML(string(content))
	val.Validate()
	for _, diag := range *val.Diagnostics {
		if diag.Message == "Orb is unused" {
			t.Errorf("Got orb is unused diagnostic")
		}
	}
}

func TestLocalOrbUsedPartsFalsePositive(t *testing.T) {
	fileURI := uri.File("some-uri")
	context := testHelpers.GetDefaultLsContext()
	content, err := os.ReadFile("./testdata/orbs/local-orb-used-parts.yml")
	assert.Nil(t, err)

	doc, err := parser.ParseFromContent(content, context, fileURI, protocol.Position{})
	assert.Nil(t, err)

	val := Validate{
		APIs: ValidateAPIs{
			DockerHub: dockerhub.NewAPI(),
		},
		Diagnostics: &[]protocol.Diagnostic{},
		Cache:       utils.CreateCache(),
		Doc:         doc,
		Context:     context,
	}
	val.Validate()
	assert.Len(t, *val.Diagnostics, 0)
}

func TestLocalOrbUnusedPartsFalseNegative(t *testing.T) {
	fileURI := uri.File("some-uri")
	context := testHelpers.GetDefaultLsContext()
	content, err := os.ReadFile("./testdata/orbs/local-orb-unused-parts.yml")
	assert.Nil(t, err)

	doc, err := parser.ParseFromContent(content, context, fileURI, protocol.Position{})
	assert.Nil(t, err)

	val := Validate{
		APIs: ValidateAPIs{
			DockerHub: dockerhub.NewAPI(),
		},
		Diagnostics: &[]protocol.Diagnostic{},
		Cache:       utils.CreateCache(),
		Doc:         doc,
		Context:     context,
	}
	val.Validate()
	messages := getDiagnosticMessages(val.Diagnostics)
	assert.ElementsMatch(t, []string{"Orb is unused", "Job is unused", "Command is unused"}, messages)
}
