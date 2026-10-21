package parser_test

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestParseForCompileStepsArgument(t *testing.T) {
	content := []byte(`version: 2.1

commands:
  wrap:
    parameters:
      steps:
        type: steps
    steps:
      - steps: << parameters.steps >>

jobs:
  build:
    docker:
      - image: cimg/base:current
    parameters:
      steps:
        type: steps
        default: []
    steps:
      - wrap:
          steps:
            - checkout
            - when:
                condition: << pipeline.git.branch >>
                steps:
                  - run: echo when
                  - unless:
                      condition: false
                      steps: [greet]
      - steps: << parameters.steps >>
  flow:
    docker:
      - image: cimg/base:current
    steps:
      - wrap:
          steps:
            - {run: echo}
            - {when: {condition: true, steps: [greet]}}
            - checkout

workflows:
  main:
    jobs:
      - build:
          steps:
            - checkout
            - with_tool_cache:
                condition: true
                steps:
                  - run: echo cached
`)
	doc, err := parser.ParseForCompile(content, "config.yml", testHelpers.DefaultSettings())
	assert.NilError(t, err)
	defer doc.Close()
	info := doc.Compile()
	assert.Assert(t, info != nil)

	t.Run("a command step's argument keeps its conditions", func(t *testing.T) {
		wrap, ok := doc.Jobs["build"].Steps[0].(ast.NamedStep)
		assert.Assert(t, ok)
		nodes, ok := info.Steps[wrap.Parameters["steps"].ValueRange]
		assert.Assert(t, ok)
		assert.Check(t, cmp.DeepEqual(describeSteps(nodes), []string{
			"checkout",
			"when(<< pipeline.git.branch >>)[run unless(false)[greet]]",
		}))
	})

	t.Run("flow mapping items in an argument are kept", func(t *testing.T) {
		wrap, ok := doc.Jobs["flow"].Steps[0].(ast.NamedStep)
		assert.Assert(t, ok)
		nodes, ok := info.Steps[wrap.Parameters["steps"].ValueRange]
		assert.Assert(t, ok)
		assert.Check(t, cmp.DeepEqual(describeSteps(nodes), []string{
			"run",
			"when(true)[greet]",
			"checkout",
		}))
	})

	t.Run("a job invocation's argument keeps its conditions", func(t *testing.T) {
		build := doc.Workflows["main"].JobInvocations[0]
		nodes, ok := info.Steps[build.Parameters["steps"].ValueRange]
		assert.Assert(t, ok)
		assert.Check(t, cmp.DeepEqual(describeSteps(nodes), []string{
			"checkout",
			"with_tool_cache(true)[run]",
		}))
	})
}
