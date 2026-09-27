package documentSymbols

import (
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

// symbolPaths flattens a symbol tree into "Parent > Child" paths, listing
// separately the paths of symbols with a blank name.
func symbolPaths(symbols []protocol.DocumentSymbol, parent []string) (paths, blank []string) {
	for _, symbol := range symbols {
		path := append(append([]string{}, parent...), symbol.Name)
		joined := strings.Join(path, " > ")
		paths = append(paths, joined)

		if strings.TrimSpace(symbol.Name) == "" {
			blank = append(blank, joined)
		}

		childPaths, childBlank := symbolPaths(symbol.Children, path)
		paths = append(paths, childPaths...)
		blank = append(blank, childBlank...)
	}

	return paths, blank
}

func TestSymbolsForDocument_HalfTypedNamesAreDropped(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		// A sibling of the half-typed node that must still be reported.
		want string
	}{
		{
			name: "step without a name",
			yaml: `version: 2.1
jobs:
  build:
    steps:
      - checkout
      -
`,
			want: "Jobs > build > Steps > checkout",
		},
		{
			name: "workflow job without a name",
			yaml: `version: 2.1
workflows:
  main:
    jobs:
      - build
      -
`,
			want: "Workflows > main > Jobs > build",
		},
		{
			name: "machine executor without an image",
			yaml: `version: 2.1
executors:
  vm:
    machine:
      image:
`,
			want: "Executors > vm",
		},
		{
			name: "job without a name",
			yaml: `version: 2.1
jobs:
  build:
    steps: [checkout]
  :
    steps: [checkout]
`,
			want: "Jobs > build",
		},
		{
			name: "orb without a name",
			yaml: `version: 2.1
orbs:
  node: circleci/node@5
  : circleci/github-cli@2
`,
			want: "Orbs > node",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := parseDoc(t, tt.yaml)
			symbols := SymbolsForDocument(&doc)
			paths, blank := symbolPaths(symbols, nil)

			assert.Check(t, cmp.Len(blank, 0))
			assert.Check(t, cmp.Contains(paths, tt.want))
		})
	}
}
