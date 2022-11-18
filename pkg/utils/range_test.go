package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
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

	assert.Equal(t, expected, actual)
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

	assert.Equal(t, expected, actual)
}
