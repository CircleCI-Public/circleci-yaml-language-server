package acceptance

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

// flowStyleConfig writes its steps, a step, its workflow jobs and a job's
// arguments in flow style, on one line. The command it defines is called only
// from a flow sequence.
const flowStyleConfig = `version: 2.1

commands:
  greet:
    steps:
      - run: echo hello

jobs:
  build:
    machine:
      image: ubuntu-2404:current
    steps: [checkout, greet]
  lint:
    machine:
      image: ubuntu-2404:current
    steps:
      - {run: make lint}
  deploy:
    machine:
      image: ubuntu-2404:current
    steps:
      - run: make deploy

workflows:
  check:
    jobs: [build, lint]
  release:
    jobs:
      - build
      - lint
      - deploy: {requires: [build, lint], filters: {branches: {only: main}}}
`

// nestedStepsConfig anchors a list of steps in one job and gives it as a
// single step of another, which the compiler flattens into the list it's in.
const nestedStepsConfig = `version: 2.1

jobs:
  build:
    machine:
      image: ubuntu-2404:current
    steps:
      - checkout
      - &setup
        - run: echo installing
        - run: echo configuring
      - run: make
  package:
    machine:
      image: ubuntu-2404:current
    steps:
      - checkout
      - *setup
      - run: make package

workflows:
  main:
    jobs:
      - build
      - package
`

// quotedJobNamesConfig names its workflow jobs in quotes, with and without
// arguments.
const quotedJobNamesConfig = `version: 2.1

jobs:
  lint:
    machine:
      image: ubuntu-2404:current
    steps:
      - run: make lint
  docs:
    machine:
      image: ubuntu-2404:current
    steps:
      - run: make docs

workflows:
  main:
    jobs:
      - "lint"
      - 'docs':
          requires:
            - "lint"
`

// pipelineValueNamesConfig names a job invocation and a matrix alias with a
// pipeline parameter, and requires them by the names they expand to.
const pipelineValueNamesConfig = `version: 2.1

parameters:
  target:
    type: enum
    enum: [staging, release]
    default: staging

jobs:
  build:
    parameters:
      os:
        type: string
        default: linux
    machine:
      image: ubuntu-2404:current
    steps:
      - run: echo building for << parameters.os >>
  publish:
    machine:
      image: ubuntu-2404:current
    steps:
      - run: echo publishing

workflows:
  main:
    jobs:
      - build:
          name: build-<< pipeline.parameters.target >>
      - build:
          matrix:
            alias: platforms-<< pipeline.parameters.target >>
            parameters:
              os: [linux, macos]
      - publish:
          requires:
            - build-release
            - platforms-release
`

// sameJobTwiceConfig invokes a job twice without a name, the second
// requiring the first by the name they share.
const sameJobTwiceConfig = `version: 2.1

jobs:
  migrate:
    machine:
      image: ubuntu-2404:current
    steps:
      - run: echo migrating

workflows:
  main:
    jobs:
      - migrate
      - migrate:
          requires:
            - migrate
`

// jobGroupConfig invokes a job group twice, once under the group's own name
// and once under another.
const jobGroupConfig = `version: 2.1

jobs:
  deploy:
    machine:
      image: ubuntu-2404:current
    steps:
      - run: echo deploying

job-groups:
  deploy-group:
    jobs:
      - deploy

workflows:
  main:
    jobs:
      - deploy-group
      - deploy-group:
          name: prod-deploy
`

// setupConfig is a setup config whose only workflow is named like its job.
const setupConfig = `version: 2.1

setup: true

jobs:
  setup:
    machine:
      image: ubuntu-2404:current
    steps:
      - checkout
      - run: echo generating config

workflows:
  setup:
    jobs:
      - setup
`

// noWorkflowsConfig has no workflows, so the compiler runs its build job in
// a workflow of its own.
const noWorkflowsConfig = `version: 2.1

jobs:
  build:
    machine:
      image: ubuntu-2404:current
    steps:
      - checkout
      - run: make
`

// prePostStepsConfig gives a workflow job pre-steps and post-steps, calling a
// command with an argument and a built-in step with none.
const prePostStepsConfig = `version: 2.1

commands:
  notify:
    parameters:
      channel:
        type: string
    steps:
      - run: echo notifying << parameters.channel >>

jobs:
  build:
    machine:
      image: ubuntu-2404:current
    steps:
      - run: make

workflows:
  main:
    jobs:
      - build:
          pre-steps:
            - checkout
            - run: echo starting
          post-steps:
            - notify:
                channel: builds
`

// commandAndJobConfig has a command and a job of the same name, each taking
// its own parameter: the step takes the command's, and the workflow job the
// job's.
const commandAndJobConfig = `version: 2.1

commands:
  build:
    parameters:
      target:
        type: string
    steps:
      - run: make << parameters.target >>

jobs:
  build:
    parameters:
      flavour:
        type: string
    machine:
      image: ubuntu-2404:current
    steps:
      - build:
          target: << parameters.flavour >>

workflows:
  main:
    jobs:
      - build:
          flavour: release
`

// nestedStepsArgumentConfig calls a command only inside the steps another
// command is given as an argument, nested inside one more.
const nestedStepsArgumentConfig = `version: 2.1

commands:
  with-retries:
    parameters:
      body:
        type: steps
    steps:
      - run: echo retrying
      - steps: << parameters.body >>
  install:
    steps:
      - run: echo installing
  cleanup:
    steps:
      - run: echo cleaning up

jobs:
  build:
    machine:
      image: ubuntu-2404:current
    steps:
      - with-retries:
          body:
            - install
            - with-retries:
                body:
                  - cleanup

workflows:
  main:
    jobs:
      - build
`

// nullMatrixParameterConfig gives a required parameter as null in a matrix,
// which makes no jobs.
const nullMatrixParameterConfig = `version: 2.1

jobs:
  smoke:
    parameters:
      version:
        type: string
    machine:
      image: ubuntu-2404:current
    steps:
      - run: echo checking << parameters.version >>

workflows:
  main:
    jobs:
      - smoke:
          matrix:
            parameters:
              version:
`

// TestWorkflowsAndStepsTheCompilerAccepts opens configs shaped like ones the
// config survey found wrongly reported, and checks each gets only what's
// really wrong with it.
func TestWorkflowsAndStepsTheCompilerAccepts(t *testing.T) {
	fake := linkedProjectFake(t)

	cases := []struct {
		name   string
		config string
		want   []string
	}{
		{name: "steps, jobs and arguments written in flow style", config: flowStyleConfig, want: []string{}},
		{name: "an anchored list of steps given as one step", config: nestedStepsConfig, want: []string{}},
		{name: "workflow jobs named in quotes", config: quotedJobNamesConfig, want: []string{}},
		{name: "requires naming jobs named with a pipeline parameter", config: pipelineValueNamesConfig, want: []string{}},
		{name: "a job invoked twice requiring itself by name", config: sameJobTwiceConfig, want: []string{}},
		{name: "a job group invoked once without a name among named ones", config: jobGroupConfig, want: []string{}},
		{name: "a setup workflow named like its job", config: setupConfig, want: []string{}},
		{name: "a build job and no workflows", config: noWorkflowsConfig, want: []string{}},
		{name: "a workflow job's pre-steps and post-steps", config: prePostStepsConfig, want: []string{}},
		{name: "a command and a job of the same name", config: commandAndJobConfig, want: []string{
			`The name "build" is already used to define a command. You might want to use a different name to avoid confusion.`,
			`The name "build" is already used to define a job. You might want to use a different name to avoid confusion.`,
		}},
		{name: "a command used only in a steps argument", config: nestedStepsArgumentConfig, want: []string{}},
		{name: "a null matrix parameter", config: nullMatrixParameterConfig, want: []string{
			"Matrix parameter 'version' is null; the matrix will produce 0 jobs",
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			session := start(t, fake, c.config, testToken)
			diagnostics := session.open(t, c.config)

			assert.Check(t, cmp.DeepEqual(diagnostics, c.want))
		})
	}
}
