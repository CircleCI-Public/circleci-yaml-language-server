package utils

import (
	"reflect"
	"testing"

	"go.lsp.dev/protocol"
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
			if got := IndexToPos(tt.args.index, tt.args.content); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("IndexToPos() = %v, want %v", got, tt.want)
			}
		})
	}
}
