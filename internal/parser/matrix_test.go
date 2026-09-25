package parser

import (
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestMatrixMemberNames(t *testing.T) {
	invocation := func(t *testing.T, jobInvocation string) (alias string, names []string) {
		t.Helper()

		content := `version: 2.1
workflows:
  main:
    jobs:
` + jobInvocation
		doc, err := ParseFromContent([]byte(content), testHelpers.DefaultSettings(), uri.File(""), protocol.Position{})
		assert.NilError(t, err)
		invocations := doc.Workflows["main"].JobInvocations
		assert.Assert(t, cmp.Len(invocations, 1))
		return invocations[0].MatrixAlias, invocations[0].MatrixNames
	}

	t.Run("the job name and each value, in declared order", func(t *testing.T) {
		alias, names := invocation(t, `      - test:
          matrix:
            parameters:
              os: [linux, windows]
              go: ["1.25", "1.26"]
`)
		assert.Check(t, cmp.Equal(alias, "test"))
		assert.Check(t, cmp.DeepEqual(names, []string{
			"test-linux-1.25", "test-linux-1.26", "test-windows-1.25", "test-windows-1.26",
		}))
	})

	t.Run("a name template, expanded for each member", func(t *testing.T) {
		_, names := invocation(t, `      - test:
          name: test-<< matrix.os >>
          matrix:
            parameters:
              os:
                - linux
                - windows
`)
		assert.Check(t, cmp.DeepEqual(names, []string{"test-linux", "test-windows"}))
	})

	t.Run("a literal name, numbered when there are several members", func(t *testing.T) {
		_, names := invocation(t, `      - test:
          name: unit
          matrix:
            parameters:
              os: [linux, windows]
`)
		assert.Check(t, cmp.DeepEqual(names, []string{"unit-1", "unit-2"}))
	})

	t.Run("a literal name, kept when exclude leaves one member", func(t *testing.T) {
		_, names := invocation(t, `      - test:
          name: unit
          matrix:
            parameters:
              os: [linux, windows]
            exclude:
              - os: windows
`)
		assert.Check(t, cmp.DeepEqual(names, []string{"unit"}))
	})

	t.Run("a matrix parameter called name", func(t *testing.T) {
		_, names := invocation(t, `      - test:
          matrix:
            parameters:
              name: [first, second]
`)
		assert.Check(t, cmp.DeepEqual(names, []string{"first", "second"}))
	})

	t.Run("excluded combinations and an alias", func(t *testing.T) {
		alias, names := invocation(t, `      - test:
          matrix:
            alias: all-tests
            parameters:
              os: [linux, windows]
              arch: [amd64, arm64]
            exclude:
              - os: windows
                arch: arm64
`)
		assert.Check(t, cmp.Equal(alias, "all-tests"))
		assert.Check(t, cmp.DeepEqual(names, []string{"test-linux-amd64", "test-linux-arm64", "test-windows-amd64"}))
	})
}
