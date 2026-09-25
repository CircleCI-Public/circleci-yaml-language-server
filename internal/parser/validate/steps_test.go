package validate

import (
	_ "embed"
	"testing"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
)

var (
	//go:embed testdata/valid_checkout_method.yml
	validCheckoutMethodYml string

	//go:embed testdata/invalid_checkout_method.yml
	invalidCheckoutMethodYml string

	//go:embed testdata/invalid_checkout_method_shallow.yml
	invalidCheckoutMethodShallowYml string

	//go:embed testdata/valid_checkout_method_shallow.yml
	validCheckoutMethodShallowYml string
)

func TestStepsValidation(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "Ignore workflow's jobs that are come from uncheckable orbs",
			YamlContent: `version: 2.1

parameters:
  dev-orb-version:
    type: string
    default: "dev:alpha"

orbs:
  ccc: cci-dev/ccc@<<pipeline.parameters.dev-orb-version>>

jobs:
  job:
    docker:
      - image: node:latest
    steps:
      - ccc/step

workflows:
  someworkflow:
    jobs:
      - job
`,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name: "Valid usage of auto-rerun fields with proper combinations",
			YamlContent: `version: 2.1

jobs:
  test-job:
    docker:
      - image: cimg/base:stable
    steps:
      - run:
          name: "Background task (valid)"
          command: "sleep 30"
          background: true
      - run:
          name: "Non-background task with max_auto_reruns only (valid)"
          command: "echo test1"
          max_auto_reruns: 3
      - run:
          name: "Non-background task with both auto-rerun fields (valid)"
          command: "echo test2"
          max_auto_reruns: 2
          auto_rerun_delay: 4m

workflows:
  test-workflow:
    jobs:
      - test-job
`,
			OnlyErrors:  true,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name: "setup_remote_docker ignores a resource_class",
			YamlContent: `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - setup_remote_docker:
          resource_class: large
          prefer_same_region: true

workflows:
  main:
    jobs:
      - build
`,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Warning(protocol.Range{
					Start: protocol.Position{Line: 8, Character: 10},
					End:   protocol.Position{Line: 8, Character: 31},
				}, "setup_remote_docker has no resource_class option, so this is ignored."),
			},
		},
	}

	CheckYamlErrors(t, testCases)
}

func TestYamlDocument_parseCheckout(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name:        "Specifying checkout method full does not result in an error",
			YamlContent: validCheckoutMethodYml,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name:        "Specifying an invalid checkout method results in an error",
			YamlContent: invalidCheckoutMethodYml,
			Diagnostics: []protocol.Diagnostic{
				{
					Severity: protocol.DiagnosticSeverityError,
					Range: protocol.Range{
						Start: protocol.Position{Line: 7, Character: 8},
						End:   protocol.Position{Line: 7, Character: 16},
					},
					Message: protocol.String("Checkout method 'invalid' is invalid"),
				},
				{
					Severity: protocol.DiagnosticSeverityError,
					Range: protocol.Range{
						Start: protocol.Position{Line: 7, Character: 8},
						End:   protocol.Position{Line: 7, Character: 16},
					},
					Message: protocol.String("Checkout depth can only be used with the shallow checkout method"),
				},
			},
		},
		{
			Name:        "Specifying checkout method shallow with depth does not result in an error",
			YamlContent: validCheckoutMethodShallowYml,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name:        "Specifying checkout method shallow without depth results in an error",
			YamlContent: invalidCheckoutMethodShallowYml,
			Diagnostics: []protocol.Diagnostic{
				{
					Severity: protocol.DiagnosticSeverityError,
					Range: protocol.Range{
						Start: protocol.Position{Line: 7, Character: 8},
						End:   protocol.Position{Line: 7, Character: 16},
					},
					Message: protocol.String("Checkout depth is not an integer"),
				},
			},
		},
		{
			Name: "A checkout method and depth from parameters are not checked",
			YamlContent: `version: 2.1

jobs:
  build:
    parameters:
      method:
        type: string
        default: full
      depth:
        type: integer
        default: 1
    docker:
      - image: cimg/base:stable
    steps:
      - checkout:
          method: << parameters.method >>
          depth: << parameters.depth >>
      - checkout:
          method: shallow
          depth: << parameters.depth >>

workflows:
  main:
    jobs:
      - build
`,
			Diagnostics: []protocol.Diagnostic{},
		},
	}
	CheckYamlErrors(t, testCases)
}

func TestCommandAndJobWithTheSameName(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "A step takes the command's parameters and a workflow job the job's",
			YamlContent: `version: 2.1

commands:
  build:
    parameters:
      cmdparam:
        type: string
    steps:
      - run: echo << parameters.cmdparam >>

jobs:
  build:
    parameters:
      jobparam:
        type: string
    docker:
      - image: cimg/base:stable
    steps:
      - build:
          cmdparam: x

workflows:
  w:
    jobs:
      - build:
          jobparam: y
`,
			OnlyErrors: true,
		},
		{
			Name: "The same holds for an orb's command and job",
			YamlContent: `version: 2.1

orbs:
  my-orb:
    commands:
      build:
        parameters:
          cmdparam:
            type: string
        steps:
          - run: echo << parameters.cmdparam >>
    jobs:
      build:
        parameters:
          jobparam:
            type: string
        docker:
          - image: cimg/base:stable
        steps:
          - run: echo << parameters.jobparam >>

jobs:
  j:
    docker:
      - image: cimg/base:stable
    steps:
      - my-orb/build:
          cmdparam: x

workflows:
  w:
    jobs:
      - j
      - my-orb/build:
          jobparam: y
`,
			OnlyErrors: true,
		},
	}

	CheckYamlErrors(t, testCases)
}

func TestStepShapeWarnings(t *testing.T) {
	const greet = `version: 2.1

commands:
  greet:
    parameters:
      who:
        type: string
        default: world
    steps:
      - run: echo << parameters.who >>

`
	testCases := []ValidateTestCase{
		{
			Name: "Parameters indented level with the step name are ignored",
			YamlContent: greet + `jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - greet:
        who: me

workflows:
  main:
    jobs:
      - build
`,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Warning(protocol.Range{
					Start: protocol.Position{Line: 16, Character: 8},
					End:   protocol.Position{Line: 16, Character: 13},
				}, "Step 'greet' has 1 sibling key(s) at the same indentation (who). Keeping 'greet' "+
					"as a no-argument step and ignoring the rest; this usually means the step's "+
					"parameters are indented to the same level as the step name."),
			},
		},
		{
			Name: "A flattened step is still looked up",
			YamlContent: `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - missing:
        who: me

workflows:
  main:
    jobs:
      - build
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 7, Character: 8},
					End:   protocol.Position{Line: 7, Character: 15},
				}, "Cannot find declaration for step missing"),
			},
		},
		{
			Name: "A step with a null body in pre-steps is a bare invocation",
			YamlContent: greet + `jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - greet

workflows:
  main:
    jobs:
      - build:
          pre-steps:
            - greet:
`,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Warning(protocol.Range{
					Start: protocol.Position{Line: 23, Character: 14},
					End:   protocol.Position{Line: 23, Character: 19},
				}, "Step 'greet' has a null body; treating as a no-argument invocation. Write `- greet` instead."),
			},
		},
		{
			Name: "A command named after a built-in step shadows it",
			YamlContent: `version: 2.1

commands:
  checkout:
    steps:
      - run: git clone "$CIRCLE_REPOSITORY_URL" .

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - checkout

workflows:
  main:
    jobs:
      - build
`,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Warning(protocol.Range{
					Start: protocol.Position{Line: 3, Character: 2},
					End:   protocol.Position{Line: 3, Character: 10},
				}, "Command 'checkout' in commands.checkout shadows built-in CircleCI command 'checkout'"),
			},
		},
	}

	CheckYamlErrors(t, testCases)
}

func TestWithToolCache(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "Steps inside with_tool_cache are checked",
			YamlContent: `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - with_tool_cache:
          tool: gradle
          steps:
            - checkout
            - missing-command

workflows:
  main:
    jobs:
      - build
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 11, Character: 14},
					End:   protocol.Position{Line: 11, Character: 29},
				}, "Cannot find declaration for step missing-command"),
			},
		},
	}

	CheckYamlErrors(t, testCases)
}
