package position

import (
	"reflect"
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestIndexToPos(t *testing.T) {
	type args struct {
		index   int
		content []byte
	}
	tests := []struct {
		name string
		args args
		want protocol.Position
	}{
		{
			name: "simple",
			args: args{
				index:   0,
				content: []byte("foo"),
			},
			want: protocol.Position{Line: 0, Character: 0},
		},
		{
			name: "simple",
			args: args{
				index:   3,
				content: []byte("foo"),
			},
			want: protocol.Position{Line: 0, Character: 3},
		},
		{
			name: "simple",
			args: args{
				index:   4,
				content: []byte("foo\nbar"),
			},
			want: protocol.Position{Line: 1, Character: 0},
		},
		{
			name: "simple",
			args: args{
				index:   17,
				content: []byte("foo\nbar\nbaz\nbiz\nboo"),
			},
			want: protocol.Position{Line: 4, Character: 1},
		},
		{
			name: "simple",
			args: args{
				index: 31,
				content: []byte(`- terraform/init:
    path: "./<<parameters.environment>>"
- terraform/validate:
    path: "./<<parameters.environment>>"
- terraform/plan:
    path: "./<<parameters.environment>>"`),
			},
			want: protocol.Position{Line: 1, Character: 13},
		},
		{
			name: "Block Scalar",
			args: args{
				index: 65,
				content: []byte(`|
            curl --request POST \
              --url 'https://<< parameters.auth0-domain >>/oauth/token' \
              --header 'content-type: application/x-www-form-urlencoded' \
              --data grant_type=client_credentials \
              --data "client_id=$AUTH0_CLIENT_ID" \
              --data "client_secret=$AUTH0_CLIENT_SECRET" \
              --data 'audience=https://<< parameters.auth0-domain >>/api/v2/' \
                | jq -r .access_token > management-api-token.txt`),
			},
			want: protocol.Position{Line: 2, Character: 29},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FromIndex(tt.args.index, tt.args.content); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FromIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPosToIndex(t *testing.T) {
	content := []byte("foo\nbar\nbaz\nbiz\nboo")
	tests := []struct {
		pos  protocol.Position
		want int
	}{
		{protocol.Position{Line: 0, Character: 0}, 0},
		{protocol.Position{Line: 0, Character: 3}, 3},
		{protocol.Position{Line: 1, Character: 0}, 4},
		{protocol.Position{Line: 1, Character: 3}, 7},
		{protocol.Position{Line: 4, Character: 1}, 17},
	}
	for _, tt := range tests {
		if got := ToIndex(tt.pos, content); got != tt.want {
			t.Errorf("ToIndex(%v) = %d, want %d", tt.pos, got, tt.want)
		}
	}
}

func TestPosToIndexRoundTrip(t *testing.T) {
	content := []byte("foo\nbar\nbaz\nbiz\nboo")
	for index := 0; index < len(content); index++ {
		pos := FromIndex(index, content)
		if got := ToIndex(pos, content); got != index {
			t.Errorf("round trip failed at index %d: got %d", index, got)
		}
	}
}

func TestLinesToIndex(t *testing.T) {
	for _, content := range []string{"", "foo", "foo\nbar\n\nbaz", "foo\n", "\n\n"} {
		lines := NewLines([]byte(content))
		for line := uint32(0); line < 6; line++ {
			for character := uint32(0); character < 6; character++ {
				pos := protocol.Position{Line: line, Character: character}
				assert.Check(t, cmp.Equal(lines.ToIndex(pos), ToIndex(pos, []byte(content))), "content %q, position %v", content, pos)
			}
		}
	}
}

func TestAdvance(t *testing.T) {
	content := []byte("foo\nbar\nbaz")
	for from := 0; from <= len(content); from++ {
		for to := from; to <= len(content); to++ {
			got := Advance(FromIndex(from, content), content[from:to])
			assert.Check(t, cmp.Equal(got, FromIndex(to, content)), "from %d to %d", from, to)
		}
	}
}
