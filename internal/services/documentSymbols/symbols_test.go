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

func TestSymbolsForDocument_InSourceOrder(t *testing.T) {
	doc := parseDoc(t, `version: 2.1
parameters:
  zeta:
    type: string
    default: ""
  alpha:
    type: string
    default: ""
executors:
  small:
    docker:
      - image: cimg/base:current
  big:
    docker:
      - image: cimg/base:current
jobs:
  test:
    executor: small
    parameters:
      z:
        type: string
      a:
        type: string
    environment:
      Z: "1"
      A: "1"
    steps:
      - checkout
  build:
    executor: big
    steps:
      - checkout
`)
	symbols := SymbolsForDocument(&doc)

	names := func(symbols []protocol.DocumentSymbol) []string {
		var names []string
		for _, symbol := range symbols {
			names = append(names, symbol.Name)
		}
		return names
	}
	child := func(symbols []protocol.DocumentSymbol, name string) []protocol.DocumentSymbol {
		for _, symbol := range symbols {
			if symbol.Name == name {
				return symbol.Children
			}
		}
		t.Fatalf("no symbol %q in %v", name, names(symbols))
		return nil
	}

	t.Run("sections", func(t *testing.T) {
		got := names(symbols)
		assert.Check(t, cmp.DeepEqual(got, []string{"Version", "Pipeline Parameters", "Executors", "Jobs"}))
	})
	t.Run("pipeline parameters", func(t *testing.T) {
		got := names(child(symbols, "Pipeline Parameters"))
		assert.Check(t, cmp.DeepEqual(got, []string{"zeta", "alpha"}))
	})
	t.Run("executors", func(t *testing.T) {
		got := names(child(symbols, "Executors"))
		assert.Check(t, cmp.DeepEqual(got, []string{"small", "big"}))
	})
	jobs := child(symbols, "Jobs")
	t.Run("jobs", func(t *testing.T) {
		got := names(jobs)
		assert.Check(t, cmp.DeepEqual(got, []string{"test", "build"}))
	})
	test := child(jobs, "test")
	t.Run("a job's entries", func(t *testing.T) {
		got := names(test)
		assert.Check(t, cmp.DeepEqual(got, []string{"Executor: small", "Parameters", "Environments", "Steps"}))
	})
	t.Run("job parameters", func(t *testing.T) {
		got := names(child(test, "Parameters"))
		assert.Check(t, cmp.DeepEqual(got, []string{"z", "a"}))
	})
	t.Run("environment keys", func(t *testing.T) {
		got := names(child(test, "Environments"))
		assert.Check(t, cmp.DeepEqual(got, []string{"Z", "A"}))
	})
}

func TestSymbolsForDocument_EnvironmentKeysOnTheirOwnLines(t *testing.T) {
	doc := parseDoc(t, `version: 2.1
executors:
  small:
    docker:
      - image: cimg/base:current
    environment:
      ONE: "1"
      TWO: "2"
jobs:
  build:
    executor: small
    environment:
      THREE: "3"
      FOUR: "4"
    steps:
      - checkout
`)
	symbols := SymbolsForDocument(&doc)

	// Each key's symbol spans its line, 0-based, and selects just the key.
	source := strings.Split(string(doc.Content), "\n")
	lines := map[string]uint32{}
	selections := map[string]string{}
	var walk func([]protocol.DocumentSymbol)
	walk = func(symbols []protocol.DocumentSymbol) {
		for _, symbol := range symbols {
			if symbol.Name == "Environments" {
				for _, key := range symbol.Children {
					lines[key.Name] = key.Range.Start.Line
					selection := key.SelectionRange
					selections[key.Name] = source[selection.Start.Line][selection.Start.Character:selection.End.Character]
				}
			}
			walk(symbol.Children)
		}
	}
	walk(symbols)

	assert.Check(t, cmp.DeepEqual(lines, map[string]uint32{"ONE": 6, "TWO": 7, "THREE": 12, "FOUR": 13}))
	assert.Check(t, cmp.DeepEqual(selections, map[string]string{"ONE": "ONE", "TWO": "TWO", "THREE": "THREE", "FOUR": "FOUR"}))
}
