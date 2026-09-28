package position

import (
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestLineContentRange(t *testing.T) {
	content := `version: "1.1"
some-key:
  property: 5
	`
	actual := LineContentRange(2, []byte(content))

	expected := protocol.Range{
		Start: protocol.Position{
			Line:      2,
			Character: 2,
		},
		End: protocol.Position{
			Line:      2,
			Character: 15,
		},
	}

	assert.Check(t, cmp.DeepEqual(expected, actual))
}

func TestAllLineContentRange(t *testing.T) {
	content := `version: "1.1"
some-key:
  property: 5
	`
	actual := AllLineContentRange([]int{1, 2}, []byte(content))

	expected := []protocol.Range{
		{
			Start: protocol.Position{
				Line:      1,
				Character: 0,
			},
			End: protocol.Position{
				Line:      1,
				Character: 9,
			},
		},

		{
			Start: protocol.Position{
				Line:      2,
				Character: 2,
			},
			End: protocol.Position{
				Line:      2,
				Character: 15,
			},
		},
	}

	assert.Check(t, cmp.DeepEqual(expected, actual))
}

func TestAllLineContentRangePastTheEnd(t *testing.T) {
	// A YAML error at the end of the stream can name the line after the last.
	content := "version: 2.1\nj"

	actual := AllLineContentRange([]int{2, 5}, []byte(content))

	last := protocol.Range{
		Start: protocol.Position{Line: 1, Character: 0},
		End:   protocol.Position{Line: 1, Character: 1},
	}
	assert.Check(t, cmp.DeepEqual(actual, []protocol.Range{last, last}))
}

func TestCompare(t *testing.T) {
	pos := func(line, character uint32) protocol.Position {
		return protocol.Position{Line: line, Character: character}
	}
	tests := []struct {
		name string
		a, b protocol.Position
		want int
	}{
		{name: "same position", a: pos(2, 3), b: pos(2, 3), want: 0},
		{name: "earlier character", a: pos(2, 1), b: pos(2, 3), want: -1},
		{name: "later character", a: pos(2, 5), b: pos(2, 3), want: 1},
		{name: "earlier line", a: pos(1, 9), b: pos(2, 0), want: -1},
		{name: "later line", a: pos(3, 0), b: pos(2, 9), want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Compare(tt.a, tt.b)
			assert.Check(t, cmp.Equal(got, tt.want))
		})
	}
}

func TestAreRangeEqual(t *testing.T) {
	rng := func(startLine, endLine uint32) protocol.Range {
		return protocol.Range{
			Start: protocol.Position{Line: startLine},
			End:   protocol.Position{Line: endLine},
		}
	}
	t.Run("same range", func(t *testing.T) {
		assert.Check(t, AreRangeEqual(rng(1, 2), rng(1, 2)))
	})
	t.Run("different start", func(t *testing.T) {
		assert.Check(t, !AreRangeEqual(rng(0, 2), rng(1, 2)))
	})
	t.Run("different end", func(t *testing.T) {
		assert.Check(t, !AreRangeEqual(rng(1, 2), rng(1, 3)))
	})
}
