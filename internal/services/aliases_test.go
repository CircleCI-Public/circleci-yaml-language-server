package languageservice

import (
	"slices"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

const aliasesConfig = `version: 2.1

orbs:
  orb:
    commands:
      c:
        parameters:
          greeting:
            type: string
        steps:
          - run: echo << parameters.greeting >>
    jobs:
      build:
        machine:
          image: ubuntu-2404:current
        steps:
          - checkout

executors:
  alias-exec: real-exec
  real-exec:
    machine:
      image: ubuntu-2404:current

commands:
  renamed-c: orb/c

jobs:
  renamed-build: orb/build
  local:
    executor: alias-exec
    steps:
      - renamed-c:
          greeting: hello

workflows:
  main:
    jobs:
      - renamed-build
      - local
`

func TestAliasNavigation(t *testing.T) {
	c := cache.New()
	settings := testHelpers.DefaultSettings()
	file := uri.File("/aliases.yml")
	c.FileCache.SetFile(cache.File{TextDocument: protocol.TextDocumentItem{URI: file, Text: aliasesConfig}})

	lines := strings.Split(aliasesConfig, "\n")
	// at is the position of the text on the line, which must be unique.
	at := func(t *testing.T, line, text string) protocol.Position {
		t.Helper()
		i := slices.Index(lines, line)
		assert.Assert(t, i != -1, "no line %q", line)
		column := strings.Index(line, text)
		assert.Assert(t, column != -1, "no %q in %q", text, line)
		return protocol.Position{Line: uint32(i), Character: uint32(column)}
	}
	startLines := func(locations []protocol.Location) []uint32 {
		res := []uint32{}
		for _, location := range locations {
			res = append(res, location.Range.Start.Line)
		}
		slices.Sort(res)
		return res
	}
	definition := func(t *testing.T, pos protocol.Position) []uint32 {
		t.Helper()
		locations, err := Definition(protocol.DefinitionParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: file},
			Position:     pos,
		}}, c, settings)
		assert.NilError(t, err)
		return startLines(locations)
	}
	references := func(t *testing.T, pos protocol.Position) []uint32 {
		t.Helper()
		locations, err := References(protocol.ReferenceParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: file},
			Position:     pos,
		}}, c, settings)
		assert.NilError(t, err)
		return startLines(locations)
	}
	lineOf := func(t *testing.T, line string) uint32 {
		t.Helper()
		return at(t, line, strings.TrimSpace(line)).Line
	}

	t.Run("definition", func(t *testing.T) {
		t.Run("of a step goes to the alias it names", func(t *testing.T) {
			got := definition(t, at(t, "      - renamed-c:", "renamed-c"))
			assert.Check(t, cmp.DeepEqual(got, []uint32{lineOf(t, "  renamed-c: orb/c")}))
		})

		t.Run("of a workflow's job goes to the alias it names", func(t *testing.T) {
			got := definition(t, at(t, "      - renamed-build", "renamed-build"))
			assert.Check(t, cmp.DeepEqual(got, []uint32{lineOf(t, "  renamed-build: orb/build")}))
		})

		t.Run("of a job's executor goes to the alias it names", func(t *testing.T) {
			got := definition(t, at(t, "    executor: alias-exec", "alias-exec"))
			assert.Check(t, cmp.DeepEqual(got, []uint32{lineOf(t, "  alias-exec: real-exec")}))
		})

		t.Run("of an alias's target goes to what it names", func(t *testing.T) {
			assert.Check(t, cmp.DeepEqual(definition(t, at(t, "  renamed-c: orb/c", "orb/c")),
				[]uint32{lineOf(t, "      c:")}))
			assert.Check(t, cmp.DeepEqual(definition(t, at(t, "  renamed-build: orb/build", "orb/build")),
				[]uint32{lineOf(t, "      build:")}))
			assert.Check(t, cmp.DeepEqual(definition(t, at(t, "  alias-exec: real-exec", "real-exec")),
				[]uint32{lineOf(t, "  real-exec:")}))
		})

		t.Run("of an argument goes to the target's parameter", func(t *testing.T) {
			got := definition(t, at(t, "          greeting: hello", "greeting"))
			assert.Check(t, cmp.DeepEqual(got, []uint32{lineOf(t, "          greeting:")}))
		})
	})

	t.Run("references", func(t *testing.T) {
		t.Run("of a command's alias are the steps that call it", func(t *testing.T) {
			got := references(t, at(t, "  renamed-c: orb/c", "renamed-c"))
			assert.Check(t, cmp.DeepEqual(got, []uint32{lineOf(t, "      - renamed-c:")}))
		})

		t.Run("of a job's alias are the workflows' jobs that run it", func(t *testing.T) {
			got := references(t, at(t, "  renamed-build: orb/build", "renamed-build"))
			assert.Check(t, cmp.DeepEqual(got, []uint32{lineOf(t, "      - renamed-build")}))
		})

		t.Run("of an executor's alias are the jobs that use it", func(t *testing.T) {
			got := references(t, at(t, "  alias-exec: real-exec", "alias-exec"))
			assert.Check(t, cmp.DeepEqual(got, []uint32{lineOf(t, "    executor: alias-exec")}))
		})
	})
}
