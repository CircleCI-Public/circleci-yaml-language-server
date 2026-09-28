package documentSymbols

import (
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func symbolKinds(symbols []protocol.DocumentSymbol, parent string, kinds map[string]protocol.SymbolKind) {
	for _, symbol := range symbols {
		path := strings.TrimPrefix(parent+" > "+symbol.Name, " > ")
		kinds[path] = symbol.Kind
		symbolKinds(symbol.Children, path, kinds)
	}
}

func TestSymbolsForDocument_Kinds(t *testing.T) {
	doc := parseDoc(t, `version: 2.1
parameters:
  deploy:
    type: boolean
    default: false
orbs:
  node: circleci/node@5.0.0
executors:
  mac:
    macos:
      xcode: 15.0.0
commands:
  greet:
    parameters:
      who:
        type: string
    steps:
      - run: echo << parameters.who >>
jobs:
  build:
    docker:
      - image: cimg/base:current
    parameters:
      os:
        type: string
    environment:
      FOO: bar
    steps:
      - greet:
          who: me
workflows:
  nightly:
    triggers:
      - schedule:
          cron: "0 0 * * *"
          filters:
            branches:
              only: [main]
    jobs:
      - build
`)
	kinds := map[string]protocol.SymbolKind{}
	symbolKinds(SymbolsForDocument(&doc), "", kinds)

	t.Run("every symbol has a kind", func(t *testing.T) {
		for path, kind := range kinds {
			valid := kind >= protocol.SymbolKindFile && kind <= protocol.SymbolKindTypeParameter
			assert.Check(t, valid, "%q has kind %d", path, kind)
		}
	})

	t.Run("kinds follow what each symbol is", func(t *testing.T) {
		want := map[string]protocol.SymbolKind{
			"Version":                                   protocol.SymbolKindProperty,
			"Pipeline Parameters":                       protocol.SymbolKindNamespace,
			"Pipeline Parameters > deploy":              protocol.SymbolKindProperty,
			"Orbs":                                      protocol.SymbolKindNamespace,
			"Orbs > node":                               protocol.SymbolKindModule,
			"Executors > mac":                           protocol.SymbolKindClass,
			"Commands > greet":                          protocol.SymbolKindFunction,
			"Commands > greet > Parameters":             protocol.SymbolKindArray,
			"Jobs > build":                              protocol.SymbolKindFunction,
			"Jobs > build > Parameters":                 protocol.SymbolKindArray,
			"Jobs > build > Parameters > os":            protocol.SymbolKindProperty,
			"Jobs > build > Environments > FOO":         protocol.SymbolKindVariable,
			"Jobs > build > Steps > greet":              protocol.SymbolKindMethod,
			"Workflows > nightly":                       protocol.SymbolKindStruct,
			"Workflows > nightly > Jobs > build":        protocol.SymbolKindMethod,
			"Workflows > nightly > Triggers > schedule": protocol.SymbolKindEvent,
		}
		for path, kind := range want {
			assert.Check(t, cmp.Contains(kinds, path))
			assert.Check(t, cmp.Equal(kinds[path], kind), path)
		}
	})
}
