// Package yamltree parses YAML with tree-sitter and walks the result.
//
// Closing a tree returns its nodes to a pool for the next parse to reuse, and
// a node does not keep its tree alive: a node read after its tree is closed
// reads another tree's data. So a Tree has one owner, who closes it once
// nothing reads its nodes any more — in this server, at the end of the request
// that parsed it.
package yamltree

import (
	"fmt"
	"iter"
	"sync"
	"sync/atomic"

	sitter "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

// Language is the YAML grammar.
var Language = grammars.YamlLanguage()

var open atomic.Int64

// Tree is a parsed document.
type Tree struct {
	once  sync.Once
	inner *sitter.Tree
}

// Parse parses YAML source. The source must outlive the tree: nodes refer to
// it by byte offset.
func Parse(content []byte) *Tree {
	tree, err := sitter.NewParser(Language).Parse(content)
	if err != nil {
		// Only a grammar the runtime cannot load fails here, which is a build
		// problem rather than something a document can cause.
		panic(err)
	}
	open.Add(1)

	return &Tree{inner: tree}
}

// Open reports how many trees have been parsed and not yet closed.
func Open() int64 {
	return open.Load()
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
	t.once.Do(func() {
		t.inner.Release()
		open.Add(-1)
	})
}

// Walk visits root and everything under it, named and anonymous, depth first
// and parents before children.
func Walk(root *sitter.Node) iter.Seq[*sitter.Node] {
	return func(yield func(*sitter.Node) bool) {
		if root == nil {
			return
		}
		cursor := sitter.NewTreeCursor(root, nil)

		for {
			if !yield(cursor.CurrentNode()) {
				return
			}
			if cursor.GotoFirstChild() {
				continue
			}
			for !cursor.GotoNextSibling() {
				if !cursor.GotoParent() {
					return
				}
			}
		}
	}
}

// Query is a compiled query.
//
// Compiling a query is far more work than running it — it was most of what
// parsing a document cost — so each is compiled once, as its package is
// initialised, and run by every request after. Running a query only reads it,
// so any number of requests can run one at once.
type Query struct {
	inner *sitter.Query
}

// MustCompileQuery compiles a query, panicking if it is not valid: that is a
// mistake in the query, not something a document can cause.
func MustCompileQuery(pattern string) *Query {
	query, err := sitter.NewQuery(pattern, Language)
	if err != nil {
		panic(fmt.Sprintf("invalid tree-sitter query %q: %s", pattern, err))
	}

	return &Query{inner: query}
}

// Run runs the query over node, calling fn for each match. The captured nodes
// belong to node's tree.
func (q *Query) Run(node *sitter.Node, fn func(match *sitter.QueryMatch)) {
	// The text is only read by text predicates, which none of these queries
	// use.
	matches := q.inner.Exec(node, Language, nil)
	for match, ok := matches.NextMatch(); ok; match, ok = matches.NextMatch() {
		fn(&match)
	}
}
