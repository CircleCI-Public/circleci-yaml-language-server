package acceptance

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

// referencesByTypeConfig passes references where a value of a type is wanted.
// What a reference holds is only known once the config is compiled, so each is
// checked by its type, not by its text.
const referencesByTypeConfig = `version: 2.1

parameters:
  cluster:
    type: enum
    enum: [test, prod]
    default: test
  shards:
    type: integer
    default: 2

commands:
  notify:
    parameters:
      token:
        type: env_var_name
    steps:
      - run: echo ${<< parameters.token >>}

jobs:
  roll:
    parameters:
      cluster:
        type: enum
        enum: [test, prod]
      token-var:
        type: string
      config-path:
        type: string
      allow-ips:
        type: boolean
        default: false
    machine:
      image: ubuntu-2404:current
    parallelism: << pipeline.parameters.shards >>
    circleci_ip_ranges: << parameters.allow-ips >>
    steps:
      - checkout:
          when: << parameters.allow-ips >>
      - notify:
          token: << parameters.token-var >>
      - run: echo << parameters.cluster >> << parameters.config-path >>

workflows:
  main:
    jobs:
      - roll:
          cluster: << pipeline.parameters.cluster >>
          token-var: ROLLBAR_TOKEN
          config-path: .circleci/config/<< pipeline.parameters.cluster >>.yml
`

// settingsFromParametersConfig gives settings that are checked against a
// catalog, or looked up by name, from parameters.
const settingsFromParametersConfig = `version: 2.1

executors:
  linux-noble:
    machine:
      image: ubuntu-2404:current
  macos-ios:
    parameters:
      xcode:
        type: string
        default: 26.5.0
    macos:
      xcode: << parameters.xcode >>

commands:
  run-step:
    parameters:
      step:
        type: string
    steps:
      - << parameters.step >>

jobs:
  build:
    parameters:
      codename:
        type: string
        default: noble
      method:
        type: string
        default: full
      depth:
        type: integer
        default: 1
    executor: linux-<< parameters.codename >>
    steps:
      - checkout:
          method: << parameters.method >>
          depth: << parameters.depth >>
      - run-step:
          step: checkout
  on-given-executor:
    parameters:
      executor:
        type: executor
    executor: << parameters.executor >>
    steps:
      - checkout
  ios:
    executor: macos-ios
    resource_class: m4pro.medium
    steps:
      - checkout

workflows:
  main:
    jobs:
      - build
      - on-given-executor:
          executor: linux-noble
      - ios
`

// orbPlaceholdersConfig declares orbs whose source is only known later: one
// named whole by a pipeline parameter, and the empty map orb-tools writes the
// orb under test into.
const orbPlaceholdersConfig = `version: 2.1

parameters:
  orb:
    type: string
    default: acme/deploy@1.0.0

orbs:
  chosen: << pipeline.parameters.orb >>
  my-orb: {}

jobs:
  integration-test:
    executor: my-orb/default
    steps:
      - my-orb/greet:
          to: world
      - chosen/prepare

workflows:
  test-deploy:
    jobs:
      - integration-test
      - chosen/deploy
      - my-orb/hello:
          name: hello-test
          to: world
`

// stepsArgumentConfig passes a steps parameter a built-in, a command of the
// config's own and one of a published orb's.
const stepsArgumentConfig = `version: 2.1

orbs:
  tools: acme/tools@1.0.0

commands:
  with-cache:
    parameters:
      steps:
        type: steps
    steps:
      - restore_cache:
          key: v1-deps
      - steps: << parameters.steps >>
  prepare:
    steps:
      - run: echo prepare

jobs:
  build:
    machine:
      image: ubuntu-2404:current
    steps:
      - with-cache:
          steps:
            - checkout
            - prepare
            - tools/install

workflows:
  main:
    jobs:
      - build
`

// stepsToolsOrbSource is the published orb stepsArgumentConfig takes a
// command from.
const stepsToolsOrbSource = `version: 2.1

commands:
  install:
    steps:
      - run: echo installing
`

// anchorsUnderUnreadKeyConfig keeps steps under a top-level key the compiler
// doesn't read, only to anchor them for its jobs. Their references are checked
// in the jobs that use them.
const anchorsUnderUnreadKeyConfig = `version: 2.1

post-steps:
  - run: &check-version
      name: Check the version
      command: gcloud --version | grep -q "<< parameters.version >>"

jobs:
  check:
    parameters:
      version:
        type: string
    machine:
      image: ubuntu-2404:current
    steps:
      - run: *check-version

workflows:
  main:
    jobs:
      - check:
          version: "1.0"
`

// parameterMistakesConfig makes the mistakes the checks above must still
// catch: references to parameters that don't exist, quoted or not, an argument
// a job doesn't take, and a value that isn't a string for a string parameter.
const parameterMistakesConfig = `version: 2.1

jobs:
  build:
    parameters:
      target:
        type: string
    machine:
      image: ubuntu-2404:current
    steps:
      - run: echo << parameters.target >>
      - run: "echo << parameters.version >>"
      - run: 'echo << pipeline.parameters.name >>'

workflows:
  main:
    jobs:
      - build:
          target: { a: 1 }
          bogus: true
`

func TestParameterFalseErrors(t *testing.T) {
	fake := linkedProjectFake(t)

	t.Run("references checked by their type", func(t *testing.T) {
		session := start(t, fake, referencesByTypeConfig, testToken)
		diagnostics := session.open(t, referencesByTypeConfig)

		assert.Check(t, cmp.DeepEqual(diagnostics, []string{}))
	})

	t.Run("settings given by parameters", func(t *testing.T) {
		session := start(t, fake, settingsFromParametersConfig, testToken)
		diagnostics := session.open(t, settingsFromParametersConfig)

		assert.Check(t, cmp.DeepEqual(diagnostics, []string{}))
	})

	t.Run("orbs known only once the pipeline runs", func(t *testing.T) {
		session := start(t, fake, orbPlaceholdersConfig, testToken)
		diagnostics := session.open(t, orbPlaceholdersConfig)

		assert.Check(t, cmp.DeepEqual(diagnostics, []string{}))
	})

	t.Run("commands passed in a steps argument", func(t *testing.T) {
		fake.AddNamespace("ns-acme", "acme")
		fake.AddOrbPackage("orb-tools", "ns-acme", "acme", "tools", false, true)
		fake.AddOrbVersion("ver-tools", "orb-tools", "acme/tools", "1.0.0", stepsToolsOrbSource, "")

		session := start(t, fake, stepsArgumentConfig, testToken)
		diagnostics := session.open(t, stepsArgumentConfig)

		assert.Check(t, cmp.DeepEqual(diagnostics, []string{}))
	})

	t.Run("references under a key the compiler doesn't read", func(t *testing.T) {
		session := start(t, fake, anchorsUnderUnreadKeyConfig, testToken)
		diagnostics := session.open(t, anchorsUnderUnreadKeyConfig)

		assert.Check(t, cmp.DeepEqual(diagnostics, []string{}))
	})

	t.Run("mistakes are still reported", func(t *testing.T) {
		session := start(t, fake, parameterMistakesConfig, testToken)
		diagnostics := session.open(t, parameterMistakesConfig)

		assert.Check(t, cmp.DeepEqual(diagnostics, []string{
			"Parameter version is not defined",
			"Pipeline parameter name is not defined",
			"Parameter target for build must be a string",
			"Parameter bogus is not defined for build",
		}))
	})
}
