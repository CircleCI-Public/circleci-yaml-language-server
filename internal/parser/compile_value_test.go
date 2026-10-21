package parser_test

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestCompileValueKeepsNullKeys(t *testing.T) {
	content := []byte(`version: 2.1
jobs:
  build:
    docker: [{image: cimg/base:current}]
    steps: [checkout]
workflows:
  main:
    when:
      and:
    jobs: [build]
`)
	doc, err := parser.ParseForCompile(content, "config.yml", testHelpers.DefaultSettings())
	assert.NilError(t, err)
	defer doc.Close()

	conds := doc.Compile().WorkflowConditions["main"]
	assert.Assert(t, cmp.Len(conds, 1))
	statement, ok := conds[0].Condition.Value.(map[string]ast.ParameterValue)
	assert.Assert(t, ok, "condition is %T", conds[0].Condition.Value)
	and, found := statement["and"]
	assert.Check(t, found, "a key with no value is kept")
	assert.Check(t, cmp.Nil(and.Value))
}
