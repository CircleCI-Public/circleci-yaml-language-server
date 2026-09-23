package tsalloc_test

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/tsalloc"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/yamltree"
)

// The counter is only useful if it notices a tree left open, and forgets it
// once it is closed.
func TestTrack(t *testing.T) {
	leaked := tsalloc.Track(t)

	tree := yamltree.Parse([]byte("version: 2.1\n"))

	t.Run("counts an open tree", func(t *testing.T) {
		assert.Check(t, leaked() > 0, "an open tree holds allocations")
	})

	tree.Close()

	t.Run("counts nothing once it is closed", func(t *testing.T) {
		assert.Check(t, cmp.Equal(leaked(), int64(0)))
	})
}
