package parser

import (
	"testing"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/yamltree"
)

// rootNodeOf parses content and returns its root node, closing the tree when
// the test ends so that nodes stay readable for the whole of it.
func rootNodeOf(t *testing.T, content []byte) *sitter.Node {
	t.Helper()

	tree := yamltree.Parse(content)
	t.Cleanup(tree.Close)

	return tree.Root()
}

// permanentRootOf parses content for fixtures built where no test is at hand,
// such as package-level variables and table builders. Its trees are never
// closed: they live as long as the test binary does, which is what such
// fixtures need.
func permanentRootOf(content []byte) *sitter.Node {
	return yamltree.Parse(content).Root()
}
