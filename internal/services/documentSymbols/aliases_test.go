package documentSymbols

import (
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestAliasSymbols(t *testing.T) {
	doc := parseDoc(t, `version: 2.1
executors:
  real-exec:
    machine:
      image: ubuntu-2404:current
  my-exec: orb/e
commands:
  renamed-c: orb/c
  greet:
    steps:
      - run: echo hi
jobs:
  build: orb/build
`)
	t.Cleanup(doc.Close)

	symbols := SymbolsForDocument(&doc)
	sections := map[string]protocol.DocumentSymbol{}
	for _, symbol := range symbols {
		sections[symbol.Name] = symbol
	}

	type entry struct {
		Name, Detail string
		Kind         protocol.SymbolKind
		Line         uint32
	}
	entries := func(section string) []entry {
		res := []entry{}
		for _, child := range sections[section].Children {
			e := entry{Name: child.Name, Kind: child.Kind, Line: child.SelectionRange.Start.Line}
			if child.Detail != nil {
				e.Detail = *child.Detail
			}
			res = append(res, e)
		}
		return res
	}

	t.Run("an executor's alias is an executor, in source order", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(entries("Executors"), []entry{
			{Name: "real-exec", Detail: "Machine", Kind: ExecutorSymbol, Line: 2},
			{Name: "my-exec", Detail: "orb/e", Kind: ExecutorSymbol, Line: 5},
		}))
	})

	t.Run("a command's alias is a command", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(entries("Commands"), []entry{
			{Name: "renamed-c", Detail: "orb/c", Kind: CommandSymbol, Line: 7},
			{Name: "greet", Kind: CommandSymbol, Line: 8},
		}))
	})

	t.Run("a job's alias is a job", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(entries("Jobs"), []entry{
			{Name: "build", Detail: "orb/build", Kind: JobSymbol, Line: 12},
		}))
	})
}
