package acceptance

import (
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
)

// The configs here are shapes the config survey found the compiler accepting
// and the server reporting as errors. The survey's corpus holds private
// repositories, so these stand in for it.

// compilerShapesConfig uses keys and value shapes the compiler takes that the
// schema once rejected: experimental.observability without notify, a job group
// named with a digit first, gcp_auth on an image, code_signing on a machine,
// a string where a list is usual, and resource classes from outside the
// schema's old list.
const compilerShapesConfig = `version: 2.1

experimental:
  observability: {}

executors:
  signing:
    machine:
      image: ubuntu-2404:current
      code_signing:
        - release-bundle
    resource_class: arm.medium
  gar:
    docker:
      - image: us-docker.pkg.dev/acme-builds/images/builder:1.0
        gcp_auth:
          oidc_service_account: builder@acme-builds.iam.gserviceaccount.com
          workload_identity_pool: circleci
          workload_identity_provider: circleci-oidc
  sized:
    parameters:
      size:
        type: string
        default: large
    machine:
      image: ubuntu-2404:current
    resource_class: << parameters.size >>

jobs:
  sign:
    executor: signing
    steps:
      - checkout
      - restore_cache:
          keys: v1-deps-{{ checksum "go.sum" }}
      - save_cache:
          key: v1-deps-{{ checksum "go.sum" }}
          paths: ~/go/pkg/mod
      - add_ssh_keys:
          fingerprints: SHA256:NPj4IcXxqQEKGXOghi/QbG2sohoNfvZ30JwCcdSSNM0
      - persist_to_workspace:
          root: .
          path: bin
  pull:
    executor: gar
    steps:
      - run: builder --version
  large:
    executor: sized
    steps:
      - run: make
      - persist_to_workspace:
          root: .
          paths: dist

job-groups:
  2-stage-build:
    jobs:
      - sign
      - pull

workflows:
  main:
    jobs:
      - 2-stage-build
      - large
`

// typedExpressionsConfig gives typed settings a << >> tag: parameters of
// every kind, and a whole expression, which takes its result's type.
const typedExpressionsConfig = `version: 2.1

parameters:
  reruns:
    type: integer
    default: 2
  delay:
    type: string
    default: 30s

jobs:
  test:
    parameters:
      split:
        type: integer
        default: 2
      fixed_ips:
        type: boolean
        default: false
      timeout:
        type: string
        default: 20m
    machine:
      image: ubuntu-2404:current
    parallelism: << parameters.split >>
    circleci_ip_ranges: << parameters.fixed_ips >>
    steps:
      - run:
          command: make test
          max_auto_reruns: << pipeline.parameters.reruns >>
          auto_rerun_delay: << pipeline.parameters.delay >>
          no_output_timeout: << parameters.timeout >>
      - run:
          command: make lint
          no_output_timeout: 1800
      - store_test_results:
          path: test-results
  shard:
    machine:
      image: ubuntu-2404:current
    parallelism: << pipeline.git.branch == "main" and 10 or 1 >>
    steps:
      - run: make shard

workflows:
  main:
    max_auto_reruns: << pipeline.parameters.reruns >>
    jobs:
      - test:
          matrix:
            parameters:
              split: [2, 4]
      - shard
`

// builtInStepsConfig passes built-in steps options the compiler ignores or
// takes, keeps caches for the compiler's longest, and gives a shell as a
// list: all of which compile, with a warning where the compiler gives one.
const builtInStepsConfig = `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:current
    shell:
      - /bin/bash
      - -eo
      - pipefail
    retention:
      caches: 30d
    steps:
      - checkout
      - setup_remote_docker:
          version: default
          prefer_same_region: true
      - store_artifacts:
          path: dist
          prefix: build
  image:
    docker:
      - image: cimg/base:current
    steps:
      - setup_remote_docker:
          resource_class: large
      - run: docker build .

workflows:
  main:
    jobs:
      - build
      - image
`

// catalogConfig names machine images, a resource class and an Xcode version
// the catalog doesn't list, which the compiler doesn't check, and an Xcode
// version it does list through an anchor and an alias.
const catalogConfig = `version: 2.1

jobs:
  canary:
    machine:
      image: ubuntu-2404:edge
    resource_class: medium
    steps:
      - run: echo canary
  big:
    machine:
      image: ubuntu-2404:current
    resource_class: gpu.nvidia.medium
    steps:
      - run: echo big
  old-xcode:
    macos:
      xcode: 14.0.0
    resource_class: macos.m1.medium.gen1
    steps:
      - run: echo old
  anchored:
    macos:
      xcode: &xcode "16.4.0"
    resource_class: macos.m1.medium.gen1
    steps:
      - run: echo anchored
  aliased:
    macos:
      xcode: *xcode
    resource_class: macos.m1.medium.gen1
    steps:
      - run: echo aliased

workflows:
  main:
    jobs:
      - canary
      - big
      - old-xcode
      - anchored
      - aliased
`

// devOrbConfig is an orb's test-deploy.yml, which imports the orb under test
// at the revision being built.
const devOrbConfig = `version: 2.1

orbs:
  greeter: acme/greeter@dev:<< pipeline.git.revision >>

jobs:
  integration:
    machine:
      image: ubuntu-2404:current
    steps:
      - greeter/greet

workflows:
  test-deploy:
    jobs:
      - integration
      - greeter/hello
`

func TestSchemaFalseErrors(t *testing.T) {
	fake := linkedProjectFake(t)
	fake.SetMachineOfferings(fakes.MachineOfferings{
		Linux: map[string][]string{
			"medium":     {"ubuntu-2404:current"},
			"large":      {"ubuntu-2404:current"},
			"arm.medium": {"ubuntu-2404:current"},
		},
		MacOS: map[string][]string{"macos.m1.medium.gen1": {"16.4.0"}},
	})

	sorted := cmpopts.SortSlices(func(a, b string) bool { return a < b })

	// open is what the server says about a config, and which of it is an
	// error, since several of these were errors that are now warnings.
	open := func(t *testing.T, config string) (said, errors []string) {
		t.Helper()

		session := start(t, fake, config, testToken)
		session.dockerHub.AddRepository("cimg", "base")
		session.dockerHub.AddTag("cimg", "base", "current", "active")

		err := session.client.DidOpen(session.workspace.URI(), config)
		assert.NilError(t, err)
		diagnostics, err := session.client.WaitForDiagnostics(session.workspace.URI())
		assert.NilError(t, err)

		errors = []string{}
		for _, d := range diagnostics {
			if d.Severity == protocol.DiagnosticSeverityError {
				errors = append(errors, diagnostic.MessageText(d))
			}
		}
		return messages(diagnostics), errors
	}

	t.Run("keys and shapes the compiler takes", func(t *testing.T) {
		said, _ := open(t, compilerShapesConfig)
		assert.Check(t, cmp.DeepEqual(said, []string{}))
	})

	t.Run("typed settings given an expression", func(t *testing.T) {
		said, _ := open(t, typedExpressionsConfig)
		assert.Check(t, cmp.DeepEqual(said, []string{}))
	})

	t.Run("an orb under test at the revision being built", func(t *testing.T) {
		said, _ := open(t, devOrbConfig)
		assert.Check(t, cmp.DeepEqual(said, []string{}))
	})

	t.Run("built-in step options the compiler ignores", func(t *testing.T) {
		said, errors := open(t, builtInStepsConfig)
		assert.Check(t, cmp.DeepEqual(said, []string{
			"The `shell` value should be a string. See https://circleci.com/docs/reference/configuration-reference/#job-name",
			"store_artifacts has no prefix option, so this is ignored.",
			"setup_remote_docker has no resource_class option, so this is ignored.",
		}, sorted))
		assert.Check(t, cmp.DeepEqual(errors, []string{}))
	})

	t.Run("machines the catalog doesn't list", func(t *testing.T) {
		said, errors := open(t, catalogConfig)
		assert.Check(t, cmp.DeepEqual(said, []string{
			`Unknown machine image "ubuntu-2404:edge"`,
			`Unknown resource class "gpu.nvidia.medium"`,
			`Unknown Xcode version "14.0.0"`,
		}, sorted))
		assert.Check(t, cmp.DeepEqual(errors, []string{}))
	})
}
