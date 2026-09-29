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
		{
			Name: "A flow mapping given for a string parameter is reported where it's given",
			YamlContent: `version: 2.1

jobs:
  build:
    parameters:
      target:
        type: string
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo << parameters.target >>

workflows:
  main:
    jobs:
      - build:
          target: { a: 1 }
`,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 16, Character: 10},
					End:   protocol.Position{Line: 16, Character: 26},
				}, "Parameter target for build must be a string"),
			},
		},
	}

	CheckYamlErrors(t, testCases)
}

// The compiler reads a plain yes, no, on or off as a boolean, as YAML 1.1 does.
func TestYAML11BooleanArguments(t *testing.T) {
	config := func(paramType, value string) string {
		return `version: 2.1

jobs:
  build:
    parameters:
      p:
        type: ` + paramType + `
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo << parameters.p >>

workflows:
  main:
    jobs:
      - build:
          p: ` + value + `
`
	}

	testCases := []ValidateTestCase{
		{
			Name:        "given for a string parameter",
			YamlContent: config("string", "yes"),
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 16, Character: 10},
					End:   protocol.Position{Line: 16, Character: 16},
				}, "Parameter p for build must be a string. `yes` is read as a boolean; quote it if it's meant as text."),
			},
		},
		{
			Name:        "quoted for a string parameter",
			YamlContent: config("string", `"yes"`),
		},
		{
			Name:        "given for a boolean parameter",
			YamlContent: config("boolean", "Off"),
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
				}, "Parameter bogus_param is not defined for my-deploy"),
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
				}, "Parameter some_param is not defined for my-build"),
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
					Start: protocol.Position{Line: 7, Character: 21},
					End:   protocol.Position{Line: 7, Character: 39},
				}, "Parameter version is not defined"),
			},
		},
		{
			Name: "A parameter used in a quoted string is checked",
			YamlContent: `version: 2.1

jobs:
  check:
    docker:
      - image: cimg/base:stable
    steps:
      - run: "echo << parameters.version >>"
      - run: 'echo << parameters.name >>'
      - run: "echo one
          << pipeline.parameters.two >>"

workflows:
  main:
    jobs:
      - check
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 7, Character: 22},
					End:   protocol.Position{Line: 7, Character: 40},
				}, "Parameter version is not defined"),
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 8, Character: 22},
					End:   protocol.Position{Line: 8, Character: 37},
				}, "Parameter name is not defined"),
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 10, Character: 13},
					End:   protocol.Position{Line: 10, Character: 36},
				}, "Pipeline parameter two is not defined"),
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

func TestParametersInExpressions(t *testing.T) {
	CheckYamlErrors(t, []ValidateTestCase{
		{
			Name: "Undefined parameters inside an expression and a section are errors",
			YamlContent: `version: 2.1

parameters:
  deploy:
    type: boolean
    default: false

commands:
  greet:
    parameters:
      loud:
        type: boolean
        default: false
    steps:
      - run: echo << parameters.loud and pipeline.parameters.deploy >> << not parameters.quiet >>
      - run: echo <<# pipeline.parameters.verbose >>-v<</ pipeline.parameters.verbose >>

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - greet

workflows:
  main:
    jobs:
      - build
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 14, Character: 78},
					End:   protocol.Position{Line: 14, Character: 94},
				}, "Parameter quiet is not defined"),
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 15, Character: 22},
					End:   protocol.Position{Line: 15, Character: 49},
				}, "Pipeline parameter verbose is not defined"),
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 15, Character: 58},
					End:   protocol.Position{Line: 15, Character: 85},
				}, "Pipeline parameter verbose is not defined"),
			},
		},
	})
}

func TestPipelineParametersInInlineOrbs(t *testing.T) {
	notInOrb := "Pipeline parameter p is not defined in orb my_orb: an inline orb can't use the config's pipeline parameters, so pass it in as a parameter"
	testCases := []ValidateTestCase{
		{
			Name: "in an inline orb's command, job and executor",
			YamlContent: `version: 2.1
parameters:
  p:
    type: string
    default: Hello World
orbs:
  my_orb:
    commands:
      run-foo:
        steps:
          - run: echo "<< pipeline.parameters.p >>"
    executors:
      my-executor:
        docker:
          - image: org-name/image-name:<< pipeline.parameters.p >>
    jobs:
      foo-job:
        executor: my-executor
        steps:
          - run-foo
          - run: echo << pipeline.parameters.p >>
workflows:
  foo:
    jobs:
      - my_orb/foo-job
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(10, 26, 47), notInOrb),
				diagnostic.Error(span(14, 42, 63), notInOrb),
				diagnostic.Error(span(20, 25, 46), notInOrb),
			},
		},
		{
			Name: "a pipeline parameter nothing defines",
			YamlContent: `version: 2.1
orbs:
  my_orb:
    jobs:
      foo-job:
        machine: true
        steps:
          - run: echo << pipeline.parameters.nope >>
workflows:
  foo:
    jobs:
      - my_orb/foo-job
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(7, 25, 49), "Pipeline parameter nope is not defined"),
			},
		},
		{
			Name: "passed in as a parameter from the workflow",
			YamlContent: `version: 2.1
parameters:
  p:
    type: string
    default: value
orbs:
  my_orb:
    jobs:
      orb_job:
        parameters:
          p:
            type: string
        machine: true
        steps:
          - run: echo "<< parameters.p >>"
workflows:
  my_workflow:
    jobs:
      - my_orb/orb_job:
          p: << pipeline.parameters.p >>
`,
			OnlyErrors:  true,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name: "the orb's own pipeline parameter",
			YamlContent: `version: 2.1
orbs:
  my_orb:
    parameters:
      p:
        type: string
        default: value
    jobs:
      orb_job:
        machine: true
        steps:
          - run: echo "<< pipeline.parameters.p >>"
workflows:
  my_workflow:
    jobs:
      - my_orb/orb_job
`,
			OnlyErrors:  true,
			Diagnostics: []protocol.Diagnostic{},
		},
	}

	CheckYamlErrors(t, testCases)
}

func TestMatrixReferences(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "a matrix value is set in the arguments, name, requires and pre-steps of its job",
			YamlContent: `version: 2.1
jobs:
  deploy:
    parameters:
      project:
        type: string
    machine: true
    steps:
      - run: echo << parameters.project >>
  notify:
    parameters:
      env:
        type: string
    machine: true
    steps:
      - run: echo << parameters.env >>
workflows:
  deploy-all:
    jobs:
      - deploy:
          name: deploy-<< matrix.env >>
          matrix:
            parameters:
              env: [dev, prod]
          project: my-app-<< matrix.env >>
          pre-steps:
            - run: echo << matrix.env >>
      - notify:
          name: notify-<< matrix.env >>
          matrix:
            parameters:
              env: [dev, prod]
          requires:
            - deploy-<< matrix.env >>
`,
			OnlyErrors:  true,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name: "a matrix value in a job-group member's matrix",
			YamlContent: `version: 2.1
jobs:
  deploy:
    parameters:
      word:
        type: string
    machine: true
    steps:
      - run: echo << parameters.word >>
job-groups:
  deploy-all:
    jobs:
      - deploy:
          name: deploy-<< matrix.word >>
          matrix:
            parameters:
              word: [qa, staging]
workflows:
  main:
    jobs:
      - deploy-all
`,
			OnlyErrors:  true,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name: "a matrix value in a job's steps",
			YamlContent: `version: 2.1
jobs:
  deploy:
    parameters:
      env:
        type: string
    machine: true
    steps:
      - run: echo << matrix.env >>
workflows:
  deploy-all:
    jobs:
      - deploy:
          matrix:
            parameters:
              env: [dev, prod]
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(8, 21, 31), "matrix.env is only set in a workflow job with a matrix"),
			},
		},
		{
			Name: "a matrix value in a workflow job without a matrix",
			YamlContent: `version: 2.1
jobs:
  deploy:
    machine: true
    steps:
      - run: echo deploy
workflows:
  deploy-all:
    jobs:
      - deploy:
          name: deploy-<< matrix.env >>
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(10, 26, 36), "matrix.env is only set in a workflow job with a matrix"),
			},
		},
		{
			Name: "a value the matrix doesn't set",
			YamlContent: `version: 2.1
jobs:
  deploy:
    parameters:
      env:
        type: string
    machine: true
    steps:
      - run: echo << parameters.env >>
workflows:
  deploy-all:
    jobs:
      - deploy:
          name: deploy-<< matrix.region >>
          matrix:
            parameters:
              env: [dev, prod]
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(13, 26, 39), "Matrix parameter region is not defined"),
			},
		},
	}

	CheckYamlErrors(t, testCases)
}
