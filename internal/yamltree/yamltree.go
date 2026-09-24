// Package yamltree parses YAML with tree-sitter and walks the result.
//
// The official Go bindings free a tree's C memory only when it is closed, and
// a node does not keep its tree alive: a node read after its tree is closed is
// a use-after-free. So a Tree has one owner, who closes it once nothing reads
// its nodes any more — in this server, at the end of the request that parsed
// it.
package yamltree

import (
	"fmt"
	"iter"
	"sync"

	tree_sitter_yaml "github.com/tree-sitter-grammars/tree-sitter-yaml/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// Language is the YAML grammar. It is static, so it is never closed.
var Language = sitter.NewLanguage(tree_sitter_yaml.Language())

// Tree is a parsed document.
type Tree struct {
	once  sync.Once
	inner *sitter.Tree
}

// Parse parses YAML source. The source must outlive the tree: nodes refer to
// it by byte offset.
func Parse(content []byte) *Tree {
	parser := sitter.NewParser()
	defer parser.Close()

	if err := parser.SetLanguage(Language); err != nil {
		// Only a grammar built for an incompatible tree-sitter ABI fails here,
		// which is a build problem rather than something a document can cause.
		panic(err)
	}

	return &Tree{inner: parser.Parse(content, nil)}
}

// Root is the root node of the tree.
func (t *Tree) Root() *sitter.Node {
	return t.inner.RootNode()
}

// Close frees the tree. It is safe to call more than once, which matters
// because documents holding a tree are copied by value: every copy shares the
// one Tree, and whichever closes it first frees it.
func (t *Tree) Close() {
	if t == nil {
		return
	}
	t.once.Do(t.inner.Close)
}

// Walk visits root and everything under it, named and anonymous, depth first
// and parents before children.
func Walk(root *sitter.Node) iter.Seq[*sitter.Node] {
	return func(yield func(*sitter.Node) bool) {
		walk(root, yield)
	}
}

func walk(node *sitter.Node, yield func(*sitter.Node) bool) bool {
	if node == nil {
		return true
	}
	if !yield(node) {
		return false
	}
	for i := uint(0); i < node.ChildCount(); i++ {
		if !walk(node.Child(i), yield) {
			return false
		}
	}
	return true
}

// Query is a compiled query.
//
// Compiling a query is far more work than running it — it was most of what
// parsing a document cost — so each is compiled once, as its package is
// initialised, and run by every request after. Running a query only reads it,
// so any number of requests can run one at once. Like Language, it is never
// closed.
type Query struct {
	inner *sitter.Query
}

// MustCompileQuery compiles a query, panicking if it is not valid: that is a
// mistake in the query, not something a document can cause.
func MustCompileQuery(pattern string) *Query {
	query, err := sitter.NewQuery(Language, pattern)
	if err != nil {
		panic(fmt.Sprintf("invalid tree-sitter query %q: %s", pattern, err))
	}

	return &Query{inner: query}
}

// Run runs the query over node, calling fn for each match. The captured nodes
// belong to node's tree.
func (q *Query) Run(node *sitter.Node, fn func(match *sitter.QueryMatch)) {
	cursor := sitter.NewQueryCursor()
	defer cursor.Close()

	// The text is only read by text predicates, which none of these queries
	// use.
	matches := cursor.Matches(q.inner, node, nil)
	for match := matches.Next(); match != nil; match = matches.Next() {
		fn(match)
	}
}
