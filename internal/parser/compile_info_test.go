package parser_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

var compileConfig = `version: 2.1

parameters:
  deploy:
    type: boolean
    default: false

conditions:
  is-main: &is-main
    equal: [main, << pipeline.git.branch >>]

orbs:
  tools:
    jobs:
      lint:
        docker:
          - image: cimg/base:current
        steps:
          - when:
              condition: << pipeline.parameters.deploy >>
              steps:
                - run: echo orb lint

commands:
  setup:
    parameters:
      extra:
        type: steps
        default: []
    steps:
      - run: echo setup
      - when:
          condition:
            and:
              - << pipeline.parameters.deploy >>
              - equal: [main, << pipeline.git.branch >>]
          steps:
            - run: echo deploying
      - steps: << parameters.extra >>

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - checkout
      - setup:
          extra:
            - when:
                condition: true
                steps:
                  - run: echo inside a parameter
      - run:
          command: make
          teardown:
            - when:
                condition: true
                steps:
                  - run: echo teardown
      - unless:
          condition: << pipeline.parameters.deploy >>
          steps:
            - run: echo outer
            - when:
                condition: *is-main
                steps:
                  - run: echo inner
      - with_tool_cache:
          condition: true
          steps:
            - run: echo cached
      - - run: echo nested a
        - when:
            condition: false
            steps:
              - run: echo nested b
  test:
    docker:
      - image: cimg/base:current
    steps:
      - run: echo test

workflows:
  main:
    when:
      not:
        equal: [scheduled_pipeline, << pipeline.trigger_source >>]
    jobs:
      - build:
          filters:
            branches:
              only: main
              ignore: [/wip-.*/, /tmp-.*/]
            tags:
              only: /^v.*/
          pre-steps:
            - run: echo pre
          post-steps:
            - when:
                condition: true
                steps:
                  - run: echo post
      - test:
          requires: [build]
          filters: pipeline.git.branch == "main"
      - tools/lint
  nightly:
    unless: << pipeline.parameters.deploy >>
    jobs:
      - test
`

func TestParseForCompile(t *testing.T) {
	content := []byte(compileConfig)

	t.Run("ParseFromContent records nothing for the compiler", func(t *testing.T) {
		doc, err := parser.ParseFromContent(content, testHelpers.DefaultSettings(), uri.URI(""), protocol.Position{})
		assert.NilError(t, err)
		defer doc.Close()
		assert.Check(t, cmp.Nil(doc.Compile()))
	})

	t.Run("the document is the one ParseFromContent gives", func(t *testing.T) {
		want, err := parser.ParseFromContent(content, testHelpers.DefaultSettings(), uri.URI(""), protocol.Position{})
		assert.NilError(t, err)
		defer want.Close()
		got, err := parser.ParseForCompile(content, "config.yml", testHelpers.DefaultSettings())
		assert.NilError(t, err)
		defer got.Close()

		optional := cmpopts.EquateComparable(protocol.Optional[string]{}, protocol.Optional[bool]{})
		assert.Check(t, cmp.DeepEqual(got.Jobs, want.Jobs, optional))
		assert.Check(t, cmp.DeepEqual(got.Commands, want.Commands, optional))
		assert.Check(t, cmp.DeepEqual(got.Workflows, want.Workflows, optional))
		assert.Check(t, cmp.DeepEqual(got.Conditions, want.Conditions))
		assert.Check(t, cmp.DeepEqual(got.StepConditions, want.StepConditions))
		assert.Check(t, cmp.DeepEqual(got.FilterExpressions, want.FilterExpressions))
		assert.Check(t, cmp.DeepEqual(got.FlattenedSteps, want.FlattenedSteps))
		assert.Check(t, cmp.DeepEqual(*got.Diagnostics, *want.Diagnostics, optional))
	})

	doc, err := parser.ParseForCompile(content, "config.yml", testHelpers.DefaultSettings())
	assert.NilError(t, err)
	defer doc.Close()
	info := doc.Compile()
	assert.Assert(t, info != nil)

	t.Run("the source is recorded", func(t *testing.T) {
		assert.Check(t, cmp.Equal(info.Source, "config.yml"))
	})

	t.Run("conditional steps keep their own steps", func(t *testing.T) {
		rng, ok := info.JobSteps["build"]
		assert.Assert(t, ok)
		assert.Check(t, cmp.DeepEqual(describeSteps(info.Steps[rng]), []string{
			"checkout",
			"setup",
			"run",
			"unless(<< pipeline.parameters.deploy >>)[run when(map[equal:[main << pipeline.git.branch >>]])[run]]",
			"with_tool_cache(true)[run]",
			"run",
			"when(false)[run]",
		}))
	})

	t.Run("a command's steps are recorded", func(t *testing.T) {
		rng, ok := info.CommandSteps["setup"]
		assert.Assert(t, ok)
		assert.Check(t, cmp.DeepEqual(describeSteps(info.Steps[rng]), []string{
			"run",
			"when(map[and:[<< pipeline.parameters.deploy >> map[equal:[main << pipeline.git.branch >>]]]])[run]",
			"extra",
		}))
	})

	t.Run("a logic statement condition is captured", func(t *testing.T) {
		cond := info.Steps[info.CommandSteps["setup"]][1].Cond
		assert.Assert(t, cond != nil)
		assert.Check(t, cmp.Equal(cond.Kind, "when"))
		assert.Check(t, cmp.DeepEqual(plainValue(cond.Condition), map[string]any{
			"and": []any{
				"<< pipeline.parameters.deploy >>",
				map[string]any{"equal": []any{"main", "<< pipeline.git.branch >>"}},
			},
		}))
	})

	main := doc.Workflows["main"]
	build := main.JobInvocations[0]
	test := main.JobInvocations[1]

	t.Run("mapping filters are captured", func(t *testing.T) {
		filters, ok := info.Filters[build.JobInvocationRange]
		assert.Assert(t, ok)
		assert.Check(t, cmp.DeepEqual(plainValue(filters), map[string]any{
			"branches": map[string]any{
				"only":   "main",
				"ignore": []any{"/wip-.*/", "/tmp-.*/"},
			},
			"tags": map[string]any{"only": "/^v.*/"},
		}))
	})

	t.Run("expression filters are captured", func(t *testing.T) {
		filters, ok := info.Filters[test.JobInvocationRange]
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(plainValue(filters), `pipeline.git.branch == "main"`))
	})

	t.Run("pre-steps and post-steps are linked from the invocation", func(t *testing.T) {
		pre, ok := info.PreSteps[build.JobInvocationRange]
		assert.Assert(t, ok)
		assert.Check(t, cmp.DeepEqual(describeSteps(info.Steps[pre]), []string{"run"}))

		post, ok := info.PostSteps[build.JobInvocationRange]
		assert.Assert(t, ok)
		assert.Check(t, cmp.DeepEqual(describeSteps(info.Steps[post]), []string{"when(true)[run]"}))
	})

	t.Run("workflow conditions are captured", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(describeConditions(info.WorkflowConditions["main"]), []string{
			"when(map[not:map[equal:[scheduled_pipeline << pipeline.trigger_source >>]]])",
		}))
		assert.Check(t, cmp.DeepEqual(describeConditions(info.WorkflowConditions["nightly"]), []string{
			"unless(<< pipeline.parameters.deploy >>)",
		}))
	})

	t.Run("an inline orb's steps are recorded under its name", func(t *testing.T) {
		orb, ok := info.Orbs["tools"]
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(orb.Source, "config.yml"))
		rng, ok := orb.JobSteps["lint"]
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(rng.Start.Line, uint32(18)), "ranges are where the orb is in the config")
		assert.Check(t, cmp.DeepEqual(describeSteps(orb.Steps[rng]), []string{
			"when(<< pipeline.parameters.deploy >>)[run]",
		}))
	})
}

// describeSteps writes each step as its name, and each conditional step as its
// kind, condition and steps.
func describeSteps(nodes []parser.StepNode) []string {
	res := []string{}
	for _, node := range nodes {
		if node.Cond == nil {
			res = append(res, node.Step.GetName())
			continue
		}
		res = append(res, fmt.Sprintf("%s(%v)[%s]", node.Cond.Kind, plainValue(node.Cond.Condition),
			strings.Join(describeSteps(node.Cond.Steps), " ")))
	}
	return res
}

func describeConditions(conds []parser.Condition) []string {
	res := []string{}
	for _, cond := range conds {
		res = append(res, fmt.Sprintf("%s(%v)", cond.Kind, plainValue(cond.Condition)))
	}
	return res
}

// plainValue is value without its names and ranges.
func plainValue(value ast.ParameterValue) any {
	switch v := value.Value.(type) {
	case map[string]ast.ParameterValue:
		res := map[string]any{}
		for key, child := range v {
			res[key] = plainValue(child)
		}
		return res
	case []ast.ParameterValue:
		res := []any{}
		for _, child := range v {
			res = append(res, plainValue(child))
		}
		return res
	}
	return value.Value
}
