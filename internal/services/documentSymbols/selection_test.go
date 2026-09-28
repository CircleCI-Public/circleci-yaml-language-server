package documentSymbols

import (
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func TestSymbolsForDocument_SelectionRanges(t *testing.T) {
	doc := parseDoc(t, `version: 2.1
parameters:
  deploy:
    type: boolean
    default: false
orbs:
  node: circleci/node@5.0.0
executors:
  small:
    docker:
      - image: cimg/base:current
commands:
  greet:
    steps:
      - run: echo hi
jobs:
  build:
    executor: small
    parameters:
      os:
        type: string
    steps:
      - greet
workflows:
  main:
    jobs:
      - build:
          name: build-linux
          os: linux
job-groups:
  group:
    jobs:
      - build
`)
	symbols := SymbolsForDocument(&doc)

	text := func(rng protocol.Range) string {
		return string(doc.Content[position.ToIndex(rng.Start, doc.Content):position.ToIndex(rng.End, doc.Content)])
	}
	selected := map[string]string{}
	var walk func(symbols []protocol.DocumentSymbol, parent string)
	walk = func(symbols []protocol.DocumentSymbol, parent string) {
		for _, symbol := range symbols {
			path := parent + symbol.Name
			inside := position.Compare(symbol.SelectionRange.Start, symbol.Range.Start) >= 0 &&
				position.Compare(symbol.SelectionRange.End, symbol.Range.End) <= 0
			assert.Check(t, inside, "%q selects outside its range", path)
			selected[path] = text(symbol.SelectionRange)
			walk(symbol.Children, path+" > ")
		}
	}
	walk(symbols, "")

	t.Run("named symbols select their name", func(t *testing.T) {
		want := map[string]string{
			"Pipeline Parameters > deploy":          "deploy",
			"Orbs > node":                           "node",
			"Executors > small":                     "small",
			"Commands > greet":                      "greet",
			"Jobs > build":                          "build",
			"Jobs > build > Parameters > os":        "os",
			"Workflows > main":                      "main",
			"Workflows > main > Jobs > build-linux": "build-linux",
			"Job Groups > group":                    "group",
			"Job Groups > group > Jobs > build":     "build",
		}
		for path, name := range want {
			assert.Check(t, cmp.Contains(selected, path))
			assert.Check(t, cmp.Equal(selected[path], name), path)
		}
	})
}
