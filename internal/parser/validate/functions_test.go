package validate

import (
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
)

var anyOrder = cmpopts.SortSlices(func(a, b string) bool { return a < b })

func functionErrors(t *testing.T, yamlContent string) []string {
	t.Helper()

	val := CreateValidateFromYAML(yamlContent)
	val.Cache.MachineOfferingsCache.Set(testMachineOfferings())
	val.Validate()

	messages := []string{}
	for _, d := range *val.Diagnostics {
		if d.Severity == 1 {
			messages = append(messages, diagnostic.MessageText(d))
		}
	}
	return messages
}

func TestFunctions(t *testing.T) {
	t.Run("a function step and its commands are accepted", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(functionErrors(t, `version: 2.1

functions:
  setup-go: github.com/circleci-functions/setup-go@v0.5.1-684fd5b

commands:
  setup:
    parameters:
      lint-cache:
        type: boolean
        default: false
    steps:
      - setup-go:
          with:
            version-file: go.mod
            golangci-lint-cache: << parameters.lint-cache >>

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - checkout
      - setup
      - setup-go
      - setup-go/cache:
          id: go-cache
      - when:
          condition: true
          steps:
            - setup-go:
                id: go

workflows:
  main:
    jobs:
      - build
`), []string{}, anyOrder))
	})

	t.Run("a step body takes only id and with", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(functionErrors(t, `version: 2.1

functions:
  setup-go: github.com/circleci-functions/setup-go@v0.5.1

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - setup-go:
          version: "1.24"
          id: Go
      - setup-go:
          with:
            versions:
              - "1.24"

workflows:
  main:
    jobs:
      - build
`), []string{
			"Unexpected key(s) in function step 'setup-go' in job 'build': version. Pass arguments under `with`.",
			`The function step 'setup-go' in job 'build' has an invalid id: "Go"`,
			"Argument(s) in the function step 'setup-go' in job 'build' must be a single value, not a list or map: versions",
		}, anyOrder))
	})

	t.Run("ids are unique within a job", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(functionErrors(t, `version: 2.1

functions:
  setup-go: github.com/circleci-functions/setup-go@v0.5.1

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - setup-go:
          id: go
      - setup-go/cache:
          id: go

workflows:
  main:
    jobs:
      - build
`), []string{
			"The job 'build' has more than one function step with id: go",
			"The job 'build' has more than one function step with id: go",
		}, anyOrder))
	})

	t.Run("a step names at most one command", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(functionErrors(t, `version: 2.1

functions:
  setup-go: github.com/circleci-functions/setup-go@v0.5.1

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - setup-go/cache/save

workflows:
  main:
    jobs:
      - build
`), []string{
			"The function step 'setup-go/cache/save' in job 'build' names more than one command. A step runs the function or one of its commands, and nothing deeper.",
		}, anyOrder))
	})

	t.Run("only the declarations that are used are checked", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(functionErrors(t, `version: 2.1

functions:
  no-version: github.com/circleci-functions/setup-go
  bad-path: setup-go@v0.1.0
  bad-version: github.com/circleci-functions/setup-go@0.1.0
  not-a-string:
    path: github.com/circleci-functions/setup-go
  unused: nonsense

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - no-version
      - bad-path
      - bad-version
      - not-a-string

workflows:
  main:
    jobs:
      - build
`), []string{
			"Function 'bad-path' has an invalid path: \"setup-go\". Expected host.tld/org/name, as in github.com/circleci-functions/setup-go@v0.1.0-abc1234",
			"Function 'bad-version' has an invalid version: \"0.1.0\". Expected a semver tag led by 'v', as in v0.1.0 or v0.1.0-abc1234",
			"Function 'no-version' is missing an '@version' suffix, which pins the release: github.com/circleci-functions/setup-go@v0.1.0-abc1234",
			"Function 'not-a-string' must be a string, for example github.com/circleci-functions/setup-go@v0.1.0-abc1234",
		}, anyOrder))
	})

	t.Run("an alias can't be an orb, a command or a built-in step", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(functionErrors(t, `version: 2.1

orbs:
  go: circleci/go@3.0.0

functions:
  go: github.com/circleci-functions/setup-go@v0.5.1
  greet: github.com/circleci-functions/greet@v0.5.1
  checkout: github.com/circleci-functions/checkout@v0.5.1

commands:
  greet:
    steps:
      - run: echo hello

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
`), []string{
			"Function name 'checkout' is also a built-in step. Rename one of them.",
			"Function name 'go' is also an orb. Rename one of them.",
			"Function name 'greet' is also a command. Rename one of them.",
		}, anyOrder))
	})

	t.Run("a functions block that isn't a map is left alone", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(functionErrors(t, `version: 2.1

functions: &anchors
  - &image cimg/base:current

jobs:
  build:
    docker:
      - image: *image
    steps:
      - checkout

workflows:
  main:
    jobs:
      - build
`), []string{}, anyOrder))
	})
}
