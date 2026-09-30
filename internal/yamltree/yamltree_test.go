package yamltree

import (
	"slices"
	"testing"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestWalk(t *testing.T) {
	tree := Parse([]byte("a:\n  - b: [c, d]\n  - 'e'\nf: |\n  g\n# h\n"))
	t.Cleanup(tree.Close)

	var want []uintptr
	var visit func(node *sitter.Node)
	visit = func(node *sitter.Node) {
		want = append(want, node.Id())
		for i := uint(0); i < node.ChildCount(); i++ {
			visit(node.Child(i))
		}
	}
	visit(tree.Root())

	t.Run("visits every node, parents first", func(t *testing.T) {
		var got []uintptr
		for node := range Walk(tree.Root()) {
			got = append(got, node.Id())
		}
		assert.Check(t, cmp.DeepEqual(got, want))
	})

	t.Run("stays under the node it starts at", func(t *testing.T) {
		mapping := tree.Root().NamedChild(0).NamedChild(0).NamedChild(0)
		first := mapping.NamedChild(0)
		var got []uintptr
		for node := range Walk(first) {
			got = append(got, node.Id())
		}
		assert.Assert(t, len(got) > 1)
		assert.Check(t, cmp.Equal(got[0], first.Id()))
		assert.Check(t, !slices.Contains(got, mapping.NamedChild(1).Id()), "the next pair is not under the first")
	})

	t.Run("stops when asked", func(t *testing.T) {
		count := 0
		for range Walk(tree.Root()) {
			count++
			if count == 3 {
				break
			}
		}
		assert.Check(t, cmp.Equal(count, 3))
	})
}
