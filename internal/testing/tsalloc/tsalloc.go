// Package tsalloc counts the trees left open, so that a test can tell whether
// the code it ran closed every tree it parsed.
//
// A closed tree's nodes go back to a pool for the next parse, so a forgotten
// Close is a pool that never refills, which nothing else reports.
package tsalloc

import (
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/yamltree"
)

// Track returns a function reporting how many trees parsed since Track was
// called are still open. Tests that use it must not run in parallel with other
// code that parses, or they count each other's trees.
func Track(t testing.TB) func() int64 {
	t.Helper()

	before := yamltree.Open()

	return func() int64 {
		return yamltree.Open() - before
	}
}
