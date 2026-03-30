package validate

import (
	"os"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
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
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 24, Character: 9},
					End:   protocol.Position{Line: 24, Character: 54},
				}, "Parameter skip for build must be a string"),
			},
		},
		{
			Name:        "Parameter usage should error when param usage is different from param definition",
			YamlContent: string(wrongParamIntegerFileContent),
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 24, Character: 9},
					End:   protocol.Position{Line: 24, Character: 54},
				}, "Parameter skip for build must be a boolean"),
			},
		},
		{
			Name:        "Parameter usage should error when param usage is different from param definition",
			YamlContent: string(wrongParamBooleanFileContent),
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
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
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
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
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
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
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
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
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
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
