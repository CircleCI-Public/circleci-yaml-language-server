package validate

import (
	"strconv"
	"strings"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/codeaction"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"go.lsp.dev/protocol"
)

func TestWorkflowMaxAutoReruns(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "Valid max_auto_reruns value 1",
			YamlContent: `version: 2.1

jobs:
  hold:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "hold"

workflows:
  test-workflow:
    max_auto_reruns: 1
    jobs:
      - hold:
          type: approval`,
			OnlyErrors:  true,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name: "Valid max_auto_reruns value 3",
			YamlContent: `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "build"

workflows:
  test-workflow:
    max_auto_reruns: 3
    jobs:
      - hold:
          type: approval
      - build:
          requires: [hold]`,
		},
		{
			Name: "Valid max_auto_reruns value 5",
			YamlContent: `version: 2.1

jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"

workflows:
  test-workflow:
    max_auto_reruns: 5
    jobs:
      - test-job`,
			OnlyErrors:  true,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name: "Invalid max_auto_reruns value 0 (below minimum)",
			YamlContent: `version: 2.1

jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"

workflows:
  test-workflow:
    max_auto_reruns: 0
    jobs:
      - test-job`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 11, Character: 21},
						End:   protocol.Position{Line: 11, Character: 22},
					},
					Severity: protocol.DiagnosticSeverityError,
					Source:   protocol.NewOptional("cci-language-server"),
					Message:  protocol.String("Must be greater than or equal to 1"),
					Data:     codeaction.Data([]protocol.CodeAction{}),
				},
			},
		},
		{
			Name: "Invalid max_auto_reruns value 6 (above maximum)",
			YamlContent: `version: 2.1

jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"

workflows:
  test-workflow:
    max_auto_reruns: 6
    jobs:
      - test-job`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 11, Character: 21},
						End:   protocol.Position{Line: 11, Character: 22},
					},
					Severity: protocol.DiagnosticSeverityError,
					Source:   protocol.NewOptional("cci-language-server"),
					Message:  protocol.String("Must be less than or equal to 5"),
					Data:     codeaction.Data([]protocol.CodeAction{}),
				},
			},
		},
		{
			Name: "Invalid max_auto_reruns string value",
			YamlContent: `version: 2.1

jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "test"

workflows:
  test-workflow:
    max_auto_reruns: "invalid"
    jobs:
      - test-job`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 11, Character: 21},
						End:   protocol.Position{Line: 11, Character: 30},
					},
					Severity: protocol.DiagnosticSeverityError,
					Source:   protocol.NewOptional("cci-language-server"),
					Message:  protocol.String("Must be greater than or equal to 1"),
					Data:     codeaction.Data([]protocol.CodeAction{}),
				},
			},
		},
	}

	CheckYamlErrors(t, testCases)
}

func TestWorkflowJobsAsAFlowSequence(t *testing.T) {
	CheckYamlErrors(t, []ValidateTestCase{
		{
			Name: "Each name in a flow sequence is an invocation",
			YamlContent: `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - checkout

workflows:
  main:
    jobs: [build, missing]
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 11, Character: 18},
					End:   protocol.Position{Line: 11, Character: 25},
				}, `Cannot find declaration for job "missing"`),
			},
		},
	})
}

func TestApprovalAndMatrixWarnings(t *testing.T) {
	const deploy = `version: 2.1

jobs:
  deploy:
    parameters:
      env:
        type: string
        default: prod
    docker:
      - image: cimg/base:current
    steps:
      - checkout

workflows:
  main:
    jobs:
`
	CheckYamlErrors(t, []ValidateTestCase{
		{
			Name: "An approval job ignores other keys",
			YamlContent: deploy + `      - hold:
          type: approval
          env: prod
      - deploy:
          requires: [hold]
`,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Warning(protocol.Range{
					Start: protocol.Position{Line: 18, Character: 10},
					End:   protocol.Position{Line: 18, Character: 19},
				}, "Job <local>/hold: 'env' is not a recognized approval key and is ignored"),
			},
		},
		{
			Name: "An approval job named after a job shadows it",
			YamlContent: deploy + `      - deploy:
          type: approval
`,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Warning(protocol.Range{
					Start: protocol.Position{Line: 16, Character: 8},
					End:   protocol.Position{Line: 16, Character: 14},
				}, "'deploy' is invoked here with type: approval, which shadows the job definition of the same name; "+
					"its own steps will not run for this invocation"),
			},
		},
		{
			Name: "A matrix with a single value for every parameter",
			YamlContent: deploy + `      - deploy:
          matrix:
            parameters:
              env: [prod]
`,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Warning(protocol.Range{
					Start: protocol.Position{Line: 17, Character: 10},
					End:   protocol.Position{Line: 17, Character: 16},
				}, "This matrix is declared with a single value for every parameter, so it always "+
					"produces exactly one job. Consider not using a matrix here."),
			},
		},
		{
			Name: "A matrix brought down to one job by exclude",
			YamlContent: deploy + `      - deploy:
          matrix:
            parameters:
              env: [prod, staging]
            exclude:
              - env: staging
`,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name: "A matrix parameter that is null supplies a required parameter, and no jobs",
			YamlContent: `version: 2.1

jobs:
  deploy:
    parameters:
      env:
        type: string
    docker:
      - image: cimg/base:current
    steps:
      - checkout

workflows:
  main:
    jobs:
      - deploy:
          matrix:
            parameters:
              env: null
`,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Warning(protocol.Range{
					Start: protocol.Position{Line: 18, Character: 14},
					End:   protocol.Position{Line: 18, Character: 23},
				}, "Matrix parameter 'env' is null; the matrix will produce 0 jobs"),
			},
		},
	})
}

func TestAlwaysTrueConditions(t *testing.T) {
	const warning = "A << >> template makes this condition a logic statement that is always true, " +
		"so the comparison is not evaluated. Use a logic statement such as `equal:` " +
		"to compare a parameter value."

	CheckYamlErrors(t, []ValidateTestCase{
		{
			Name: "A template compared outside the template",
			YamlContent: `version: 2.1

parameters:
  env:
    type: string
    default: prod

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - when:
          condition: << pipeline.parameters.env >> == "prod"
          steps:
            - checkout

workflows:
  main:
    when: << pipeline.parameters.env >> != "dev"
    jobs:
      - build
`,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Warning(protocol.Range{
					Start: protocol.Position{Line: 13, Character: 21},
					End:   protocol.Position{Line: 13, Character: 60},
				}, warning),
				diagnostic.Warning(protocol.Range{
					Start: protocol.Position{Line: 19, Character: 10},
					End:   protocol.Position{Line: 19, Character: 48},
				}, warning),
			},
		},
		{
			Name: "An expression, a template alone and a logic statement",
			YamlContent: `version: 2.1

parameters:
  deploy:
    type: boolean
    default: false

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - when:
          condition: << pipeline.parameters.deploy >>
          steps:
            - checkout
      - when:
          condition:
            equal: [main, << pipeline.git.branch >>]
          steps:
            - checkout

workflows:
  main:
    when: pipeline.git.branch == "main"
    jobs:
      - build
`,
			Diagnostics: []protocol.Diagnostic{},
		},
	})
}

func TestMatrixJobLimit(t *testing.T) {
	values := func(n int) string {
		items := make([]string, n)
		for i := range items {
			items[i] = "v" + strconv.Itoa(i)
		}
		return "[" + strings.Join(items, ", ") + "]"
	}
	config := func(a, b int) string {
		return `version: 2.1

jobs:
  test:
    parameters:
      a:
        type: string
      b:
        type: string
    docker:
      - image: cimg/base:current
    steps:
      - run: echo << parameters.a >> << parameters.b >>

workflows:
  main:
    jobs:
      - test:
          matrix:
            parameters:
              a: ` + values(a) + `
              b: ` + values(b) + `
`
	}

	CheckYamlErrors(t, []ValidateTestCase{
		{
			Name:        "At the limit",
			YamlContent: config(16, 8),
			OnlyErrors:  true,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name:        "Over the limit",
			YamlContent: config(16, 9),
			OnlyErrors:  true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 18, Character: 10},
					End:   protocol.Position{Line: 18, Character: 16},
				}, "The test build matrix expands to 144 jobs. Matrices cannot generate more than 128 jobs."),
			},
		},
	})
}
