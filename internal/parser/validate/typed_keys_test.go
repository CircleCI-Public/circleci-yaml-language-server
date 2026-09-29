package validate

import (
	"testing"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
)

func TestTypedKeyReferences(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "parameters of the types the settings take",
			YamlContent: `version: 2.1
parameters:
  n:
    type: integer
    default: 2
commands:
  c:
    parameters:
      delay:
        type: string
        default: 10s
      dlc:
        type: boolean
        default: false
    steps:
      - setup_remote_docker:
          docker_layer_caching: << parameters.dlc >>
      - run:
          command: echo hi
          max_auto_reruns: 3
          auto_rerun_delay: << parameters.delay >>
jobs:
  build:
    parameters:
      n:
        type: integer
        default: 1
      ip:
        type: boolean
        default: true
      method:
        type: enum
        enum: [full, blobless]
        default: full
    machine: true
    parallelism: << pipeline.parameters.n >>
    circleci_ip_ranges: << parameters.ip >>
    steps:
      - checkout:
          method: << parameters.method >>
      - c:
          delay: 5m
      - run:
          command: echo hi
          max_auto_reruns: << parameters.n >>
workflows:
  workflow:
    jobs:
      - build:
          n: 5
          method: blobless
`,
			OnlyErrors: true,
		},
		{
			Name: "a parameter of a type the setting doesn't take",
			YamlContent: `version: 2.1
jobs:
  build:
    parameters:
      b:
        type: string
        default: "true"
      n:
        type: enum
        enum: ["1", "2"]
        default: "1"
    machine: true
    parallelism: << parameters.n >>
    circleci_ip_ranges: "<< parameters.b >>"
    steps:
      - setup_remote_docker:
          prefer_same_region: << parameters.b >>
      - checkout
workflows:
  workflow:
    jobs:
      - build
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(12, 17, 35),
					"parallelism must be an integer or an expression evaluating to an integer, but parameter n is of type enum"),
				diagnostic.Error(span(13, 24, 44),
					"circleci_ip_ranges must be a boolean, but parameter b is of type string"),
				diagnostic.Error(span(16, 30, 48),
					"setup_remote_docker: prefer_same_region must be a boolean, but parameter b is of type string"),
			},
		},
		{
			Name: "a default or an argument the setting doesn't allow",
			YamlContent: `version: 2.1
parameters:
  n:
    type: integer
    default: 0
jobs:
  build:
    parameters:
      reruns:
        type: integer
        default: 6
      method:
        type: string
        default: full
    machine: true
    parallelism: << pipeline.parameters.n >>
    steps:
      - checkout:
          method: << parameters.method >>
      - run:
          command: echo hi
          max_auto_reruns: << parameters.reruns >>
workflows:
  workflow:
    jobs:
      - build:
          method: shallowest
      - build:
          name: build-2
          matrix:
            parameters:
              reruns: [2, 0]
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(4, 4, 14),
					"parallelism must be a positive integer: parameter n is used for parallelism, and defaults to 0"),
				diagnostic.Error(span(10, 8, 18),
					"max_auto_reruns must be an integer between 1 and 5: parameter reruns is used for max_auto_reruns, and defaults to 6"),
				diagnostic.Error(span(26, 18, 28),
					`Unsupported checkout method, allowed methods are "blobless", "shallow", and "full": parameter method is used for method, and is given shallowest`),
				diagnostic.Error(span(31, 26, 27),
					"max_auto_reruns must be an integer between 1 and 5: parameter reruns is used for max_auto_reruns, and is given 0"),
			},
		},
		{
			Name: "a default used by two settings is reported once",
			YamlContent: `version: 2.1
jobs:
  build:
    parameters:
      reruns:
        type: integer
        default: 0
    machine: true
    steps:
      - run:
          command: echo hi
          max_auto_reruns: << parameters.reruns >>
      - run:
          command: echo bye
          max_auto_reruns: << parameters.reruns >>
workflows:
  workflow:
    jobs:
      - build
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(6, 8, 18),
					"max_auto_reruns must be an integer between 1 and 5: parameter reruns is used for max_auto_reruns, and defaults to 0"),
			},
		},
		{
			Name: "a command's default and the arguments its steps give it",
			YamlContent: `version: 2.1
commands:
  c:
    parameters:
      delay:
        type: string
        default: 601s
    steps:
      - run:
          command: echo hi
          max_auto_reruns: 3
          auto_rerun_delay: << parameters.delay >>
jobs:
  build:
    machine: true
    steps:
      - c:
          delay: 1m30s
      - c:
          delay: << pipeline.parameters.delay >>
workflows:
  workflow:
    jobs:
      - build:
          pre-steps:
            - c:
                delay: 5m
parameters:
  delay:
    type: string
    default: 10s
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(6, 8, 21),
					"auto_rerun_delay must not exceed 10 minutes (600 seconds): parameter delay is used for auto_rerun_delay, and defaults to 601s"),
				diagnostic.Error(span(17, 17, 22),
					"auto_rerun_delay must match ^((10|[1-9])m|([1-9][0-9]*)s)$: parameter delay is used for auto_rerun_delay, and is given 1m30s"),
			},
		},
		{
			Name: "an inline orb's command",
			YamlContent: `version: 2.1
orbs:
  inline:
    commands:
      c:
        parameters:
          delay:
            type: integer
            default: 5
        steps:
          - run:
              command: echo hi
              max_auto_reruns: 3
              auto_rerun_delay: << parameters.delay >>
jobs:
  build:
    machine: true
    steps:
      - inline/c
workflows:
  workflow:
    jobs:
      - build
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(13, 32, 54),
					"auto_rerun_delay must match ^((10|[1-9])m|([1-9][0-9]*)s)$, but parameter delay is of type integer"),
			},
		},
		{
			Name: "a machine's docker_layer_caching",
			YamlContent: `version: 2.1
parameters:
  dlc:
    type: string
    default: "yes"
executors:
  my-exec:
    parameters:
      dlc:
        type: boolean
        default: true
      text:
        type: string
        default: "true"
    machine:
      image: ubuntu-2404:current
      docker_layer_caching: << parameters.text >>
  pipeline-exec:
    machine:
      image: ubuntu-2404:current
      docker_layer_caching: << pipeline.parameters.dlc >>
  fine-exec:
    parameters:
      dlc:
        type: boolean
        default: true
    machine:
      image: ubuntu-2404:current
      docker_layer_caching: << parameters.dlc >>
jobs:
  build:
    parameters:
      dlc:
        type: enum
        enum: ["true", "false"]
        default: "true"
    machine:
      image: ubuntu-2404:current
      docker_layer_caching: << parameters.dlc >>
    steps:
      - checkout
  test:
    executor: my-exec
    steps:
      - checkout
  lint:
    executor: pipeline-exec
    steps:
      - checkout
  deploy:
    executor: fine-exec
    steps:
      - checkout
workflows:
  workflow:
    jobs:
      - build
      - test
      - lint
      - deploy
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(span(16, 28, 49),
					"Executor my-exec: docker_layer_caching must be a boolean, but parameter text is of type string"),
				diagnostic.Error(span(20, 28, 57),
					"Executor pipeline-exec: docker_layer_caching must be a boolean, but parameter dlc is of type string"),
				diagnostic.Error(span(38, 28, 48),
					"Executor <inline>: docker_layer_caching must be a boolean, but parameter dlc is of type enum"),
			},
		},
	}

	CheckYamlErrors(t, testCases)
}
