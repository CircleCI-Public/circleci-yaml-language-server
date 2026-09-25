package validate

import (
	"os"
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
)

func TestJobParameterType(t *testing.T) {
	correctParamFilePath := "./testdata/correct_param_type.yml"
	correctParamFileContent, err := os.ReadFile(correctParamFilePath)
	if err != nil {
		panic(err)
	}
	wrongParamFilePath := "./testdata/wrong_param_type.yml"
	wrongParamFileContent, err2 := os.ReadFile(wrongParamFilePath)
	if err2 != nil {
		panic(err2)
	}
	wrongParamIntegerFilePath := "./testdata/wrong_param_type_integer.yml"
	wrongParamIntegerFileContent, err2 := os.ReadFile(wrongParamIntegerFilePath)
	if err2 != nil {
		panic(err2)
	}
	wrongParamBooleanFilePath := "./testdata/wrong_param_type_boolean.yml"
	wrongParamBooleanFileContent, err2 := os.ReadFile(wrongParamBooleanFilePath)
	if err2 != nil {
		panic(err2)
	}
	testCases := []ValidateTestCase{
		{
			Name:        "Using a global Parameter on a job parameter with the same type definition should not result in error",
			YamlContent: string(correctParamFileContent),
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name:        "Parameter usage should error when param usage is different from param definition",
			YamlContent: string(wrongParamFileContent),
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 24, Character: 9},
					End:   protocol.Position{Line: 24, Character: 54},
				}, "Parameter skip for build must be a string"),
			},
		},
		{
			Name:        "Parameter usage should error when param usage is different from param definition",
			YamlContent: string(wrongParamIntegerFileContent),
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 24, Character: 9},
					End:   protocol.Position{Line: 24, Character: 54},
				}, "Parameter skip for build must be a boolean"),
			},
		},
		{
			Name:        "Parameter usage should error when param usage is different from param definition",
			YamlContent: string(wrongParamBooleanFileContent),
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 24, Character: 9},
					End:   protocol.Position{Line: 24, Character: 54},
				}, "Parameter skip for build must be a boolean"),
			},
		},
	}

	CheckYamlErrors(t, testCases)
}

func TestJobInvocationMissingRequiredParameter(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "Missing required string parameter",
			YamlContent: `version: 2.1

jobs:
  morejob:
    parameters:
      go_version:
        description: the version of Go
        type: string
    docker:
      - image: cimg/go:<<parameters.go_version>>
    steps:
      - checkout

workflows:
  test-workflow:
    jobs:
      - morejob`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 16, Character: 6},
					End:   protocol.Position{Line: 16, Character: 15},
				}, "Parameter go_version is required for morejob"),
			},
		},
		{
			Name: "Optional parameter with default is not required",
			YamlContent: `version: 2.1

jobs:
  morejob:
    parameters:
      go_version:
        type: string
        default: "1.21"
    docker:
      - image: cimg/go:<<parameters.go_version>>
    steps:
      - checkout

workflows:
  test-workflow:
    jobs:
      - morejob`,
			OnlyErrors: true,
		},
		{
			Name: "Required parameter provided - no error",
			YamlContent: `version: 2.1

jobs:
  morejob:
    parameters:
      go_version:
        type: string
    docker:
      - image: cimg/go:<<parameters.go_version>>
    steps:
      - checkout

workflows:
  test-workflow:
    jobs:
      - morejob:
          go_version: "1.21"`,
			OnlyErrors: true,
		},
		{
			Name: "Multiple params - one required missing, one optional",
			YamlContent: `version: 2.1

jobs:
  my-deploy:
    parameters:
      env:
        type: string
      verbose:
        type: boolean
        default: false
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "deploy"

workflows:
  test-workflow:
    jobs:
      - my-deploy`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 18, Character: 6},
					End:   protocol.Position{Line: 18, Character: 17},
				}, "Parameter env is required for my-deploy"),
			},
		},
	}

	CheckYamlErrors(t, testCases)
}

func TestJobInvocationUndefinedParameter(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "Undefined parameter passed to job invocation",
			YamlContent: `version: 2.1

jobs:
  my-deploy:
    parameters:
      env:
        type: string
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "deploy"

workflows:
  test-workflow:
    jobs:
      - my-deploy:
          env: prod
          bogus_param: hello`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 17, Character: 10},
					End:   protocol.Position{Line: 17, Character: 28},
				}, "Parameter bogus_param is not defined in my-deploy"),
			},
		},
		{
			Name: "No parameters defined on job but params passed",
			YamlContent: `version: 2.1

jobs:
  my-build:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "build"

workflows:
  test-workflow:
    jobs:
      - my-build:
          some_param: value`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 13, Character: 10},
					End:   protocol.Position{Line: 13, Character: 27},
				}, "Parameter some_param is not defined in my-build"),
			},
		},
	}

	CheckYamlErrors(t, testCases)
}

func TestJobInvocationMatrixParams(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "Valid matrix enum values",
			YamlContent: `version: 2.1

jobs:
  test:
    parameters:
      os:
        type: string
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo <<parameters.os>>

workflows:
  test-workflow:
    jobs:
      - test:
          matrix:
            parameters:
              os: [linux, macos]`,
			OnlyErrors: true,
		},
		{
			Name: "Matrix satisfies required parameter",
			YamlContent: `version: 2.1

jobs:
  test:
    parameters:
      version:
        type: string
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo <<parameters.version>>

workflows:
  test-workflow:
    jobs:
      - test:
          matrix:
            parameters:
              version: ["14", "16", "18"]`,
			OnlyErrors: true,
		},
	}

	CheckYamlErrors(t, testCases)
}

func TestPipelineValuesAsParameters(t *testing.T) {
	// requiredParams defines a command and a job, each taking one parameter
	// of every type a pipeline value might be given to.
	const requiredParams = `
      count:
        type: integer
        default: 1
      flag:
        type: boolean
        default: false
      text:
        type: string
        default: ""
      choice:
        type: enum
        enum: [main, other]
        default: main
`
	config := func(values string) string {
		return `version: 2.1

commands:
  cmd:
    parameters:` + requiredParams + `    steps:
      - run: echo

jobs:
  j:
    parameters:` + requiredParams + `    docker:
      - image: cimg/base:stable
    steps:
      - cmd:` + values + `

workflows:
  w:
    jobs:
      - j:` + values + `
`
	}

	testCases := []ValidateTestCase{
		{
			Name: "Pipeline values are given the type they will have",
			YamlContent: config(`
          count: << pipeline.number >>
          flag: << pipeline.git.branch.is_default >>
          text: << pipeline.number >>
          choice: << pipeline.git.branch >>`),
			OnlyErrors: true,
		},
		{
			Name: "A pipeline value that is not known is accepted",
			YamlContent: config(`
          count: << pipeline.not_yet_documented >>`),
			OnlyErrors: true,
		},
	}

	CheckYamlErrors(t, testCases)

	t.Run("A pipeline value of the wrong type is still an error", func(t *testing.T) {
		val := CreateValidateFromYAML(config(`
          count: << pipeline.id >>`))
		val.Validate()

		said := getDiagnosticMessages(val.Diagnostics)
		assert.Check(t, cmp.Contains(said, "Parameter count for cmd must be a integer"))
		assert.Check(t, cmp.Contains(said, "Parameter count for j must be a integer"))
	})
}

// A reference's value is only known once the config is compiled, so these
// are checked by type, not by value.
func TestReferencesAsParameters(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "A pipeline enum passed to a job's enum",
			YamlContent: `version: 2.1

parameters:
  cluster:
    type: enum
    enum: [test, prod]
    default: test

jobs:
  roll:
    parameters:
      cluster:
        type: enum
        enum: [test, prod]
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo << parameters.cluster >>

workflows:
  main:
    jobs:
      - roll:
          cluster: << pipeline.parameters.cluster >>
`,
			OnlyErrors:  true,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name: "A string parameter passed to an env_var_name",
			YamlContent: `version: 2.1

commands:
  notify:
    parameters:
      token:
        type: env_var_name
    steps:
      - run: echo ${<< parameters.token >>}

jobs:
  deploy:
    parameters:
      token-var:
        type: string
    docker:
      - image: cimg/base:stable
    steps:
      - notify:
          token: << parameters.token-var >>

workflows:
  main:
    jobs:
      - deploy:
          token-var: ROLLBAR_TOKEN
`,
			OnlyErrors:  true,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name: "A pipeline enum written into a string",
			YamlContent: `version: 2.1

parameters:
  driver:
    type: enum
    enum: [kubernetes, machine]
    default: kubernetes

jobs:
  trigger:
    parameters:
      path:
        type: string
    docker:
      - image: cimg/base:stable
    steps:
      - run: cat << parameters.path >>

workflows:
  main:
    jobs:
      - trigger:
          path: .circleci/config/<< pipeline.parameters.driver >>.yml
`,
			OnlyErrors:  true,
			Diagnostics: []protocol.Diagnostic{},
		},
	}

	CheckYamlErrors(t, testCases)
}

func TestParametersUnderUnreadTopLevelKeys(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "Steps kept under an unread key for their anchors",
			YamlContent: `version: 2.1

post-steps:
  - run: &check-version
      name: Check the version
      command: gcloud --version | grep -q "<< parameters.version >>"

jobs:
  check:
    parameters:
      version:
        type: string
    docker:
      - image: cimg/base:stable
    steps:
      - run: *check-version

workflows:
  main:
    jobs:
      - check:
          version: "1.0"
`,
			OnlyErrors:  true,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name: "A parameter used under jobs is still checked",
			YamlContent: `version: 2.1

jobs:
  check:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo << parameters.version >>

workflows:
  main:
    jobs:
      - check
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 7, Character: 18},
					End:   protocol.Position{Line: 7, Character: 42},
				}, "Parameter version is not defined"),
			},
		},
	}

	CheckYamlErrors(t, testCases)
}

func TestStepsParameterValues(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "Steps named by a built-in, a command, or an orb that isn't fetched",
			YamlContent: `version: 2.1

orbs:
  bp-go: https://example.com/orbs/bp-go.yml

commands:
  with-cache:
    parameters:
      steps:
        type: steps
    steps:
      - steps: << parameters.steps >>
  prepare:
    steps:
      - run: echo prepare

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - with-cache:
          steps:
            - checkout
            - prepare
            - bp-go/private-mod-init
            - missing

workflows:
  main:
    jobs:
      - build
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 26, Character: 14},
					End:   protocol.Position{Line: 26, Character: 21},
				}, "Cannot find a definition for command named missing"),
			},
		},
	}

	CheckYamlErrors(t, testCases)
}
