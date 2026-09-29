package validate

import (
	"testing"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
)

func TestExecutorFromANonStringReference(t *testing.T) {
	CheckYamlErrors(t, []ValidateTestCase{
		{
			Name: "a pipeline parameter that isn't a string",
			YamlContent: `version: 2.1
parameters:
  p:
    type: integer
    default: 3
jobs:
  build:
    executor: << pipeline.parameters.p >>
    steps:
      - run: echo hi
workflows:
  w:
    jobs:
      - build
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(7, 14, 41),
					"Executor invocation << pipeline.parameters.p >> must resolve to an executor name"),
			},
		},
		{
			Name: "a job parameter that isn't a string, given as the executor's name",
			YamlContent: `version: 2.1
executors:
  target:
    docker:
      - image: cimg/base:2024.01
jobs:
  build:
    parameters:
      q:
        type: boolean
        default: true
    executor:
      name: << parameters.q >>
    steps:
      - run: echo hi
workflows:
  w:
    jobs:
      - build
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(12, 12, 30),
					"Executor invocation << parameters.q >> must resolve to an executor name"),
			},
		},
		{
			Name: "a pipeline value that isn't a string",
			YamlContent: `version: 2.1
jobs:
  build:
    executor: << pipeline.number >>
    steps:
      - run: echo hi
workflows:
  w:
    jobs:
      - build
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(3, 14, 35),
					"Executor invocation << pipeline.number >> must resolve to an executor name"),
			},
		},
		{
			Name: "string, enum and executor parameters",
			YamlContent: `version: 2.1
parameters:
  runner:
    type: string
    default: target
executors:
  target:
    docker:
      - image: cimg/base:2024.01
jobs:
  from-pipeline:
    executor: << pipeline.parameters.runner >>
    steps:
      - run: echo hi
  from-enum:
    parameters:
      e:
        type: enum
        enum: [target]
        default: target
    executor: << parameters.e >>
    steps:
      - run: echo hi
  from-executor:
    parameters:
      e:
        type: executor
        default: target
    executor: << parameters.e >>
    steps:
      - run: echo hi
workflows:
  w:
    jobs:
      - from-pipeline
      - from-enum
      - from-executor
`,
			OnlyErrors: true,
		},
	})
}

func TestMatchesValueFromANonStringReference(t *testing.T) {
	CheckYamlErrors(t, []ValidateTestCase{
		{
			Name: "a pipeline parameter that isn't a string",
			YamlContent: `version: 2.1
parameters:
  foo:
    type: boolean
    default: true
jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - checkout
workflows:
  matches:
    when:
      matches:
        pattern: /foo.*/
        value: << pipeline.parameters.foo >>
    jobs:
      - build
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(16, 15, 44),
					"matches: value must produce a string, but the referenced parameter is not of type string"),
			},
		},
		{
			Name: "a job parameter and a pipeline value in step conditions, quoted or not",
			YamlContent: `version: 2.1
jobs:
  build:
    parameters:
      count:
        type: integer
        default: 1
      branch:
        type: string
        default: main
    docker:
      - image: cimg/base:current
    steps:
      - when:
          condition:
            matches: {pattern: "^1$", value: << parameters.count >>}
          steps:
            - checkout
      - when:
          condition:
            matches:
              pattern: "^1$"
              value: << pipeline.number >>
          steps:
            - checkout
      - when:
          condition:
            matches:
              pattern: "^main$"
              value: << parameters.branch >>
          steps:
            - checkout
      - when:
          condition:
            matches:
              pattern: "^1$"
              value: "<< parameters.count >>"
          steps:
            - checkout
      - when:
          condition:
            matches:
              pattern: "^1-.*$"
              value: << parameters.count >>-<< pipeline.git.branch >>
          steps:
            - checkout
workflows:
  w:
    jobs:
      - build
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(15, 45, 67),
					"matches: value must produce a string, but the referenced parameter is not of type string"),
				diagnostic.Error(span(22, 21, 42),
					"matches: value must produce a string, but the referenced parameter is not of type string"),
				diagnostic.Error(span(36, 21, 45),
					"matches: value must produce a string, but the referenced parameter is not of type string"),
			},
		},
	})
}

func TestExecutorArgumentFromANonStringReference(t *testing.T) {
	const jobs = `version: 2.1
executors:
  target:
    docker:
      - image: cimg/base:2024.01
jobs:
  build:
    parameters:
      e:
        type: executor
      q:
        type: integer
        default: 5
      s:
        type: string
        default: target
      other:
        type: executor
        default: target
    executor: << parameters.e >>
    steps:
      - run: echo hi
`
	CheckYamlErrors(t, []ValidateTestCase{
		{
			Name: "a parameter of the job that isn't a string",
			YamlContent: jobs + `workflows:
  w:
    jobs:
      - build:
          e:
            name: << parameters.q >>
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(27, 18, 36),
					"Executor invocation << parameters.q >> must resolve to an executor name"),
			},
		},
		{
			Name: "a pipeline value that isn't a string",
			YamlContent: jobs + `workflows:
  w:
    jobs:
      - build:
          e:
            name: << pipeline.number >>
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(27, 18, 39),
					"Executor invocation << pipeline.number >> must resolve to an executor name"),
			},
		},
		{
			Name: "a parameter of the job group member's job that isn't a string",
			YamlContent: jobs + `job-groups:
  group:
    jobs:
      - build:
          e:
            name: << parameters.q >>
workflows:
  w:
    jobs:
      - group
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(27, 18, 36),
					"Executor invocation << parameters.q >> must resolve to an executor name"),
			},
		},
		{
			Name: "parameters of the job that can name an executor",
			YamlContent: jobs + `workflows:
  w:
    jobs:
      - build:
          name: from-a-string
          e:
            name: << parameters.s >>
      - build:
          name: from-an-executor
          e:
            name: << parameters.other >>
`,
			OnlyErrors:  true,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name: "a parameter the job doesn't have",
			YamlContent: jobs + `workflows:
  w:
    jobs:
      - build:
          e:
            name: << parameters.nope >>
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(27, 21, 36), "Parameter nope is not defined"),
			},
		},
	})
}
