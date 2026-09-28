package paramref

import (
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestGetParamNameUsedAtPos(t *testing.T) {
	type args struct {
		content  []byte
		position protocol.Position
	}
	tests := []struct {
		name                string
		args                args
		wantedName          string
		wantedPipelineParam bool
	}{
		{
			name: "Simple test case 1",
			args: args{
				content: []byte(`steps:
	- run:
		- command: << parameters.command >>`),
				position: protocol.Position{
					Line:      2,
					Character: 28,
				},
			},
			wantedName:          "command",
			wantedPipelineParam: false,
		},
		{
			name: "Simple test case 2",
			args: args{
				content: []byte(`executor:
	docker: << parameters.docker >>`),
				position: protocol.Position{
					Line:      1,
					Character: 32,
				},
			},
			wantedName:          "docker",
			wantedPipelineParam: false,
		},
		{
			name: "Simple test case 3",
			args: args{
				content: []byte(`- run:
	when: << parameters.when >>`),
				position: protocol.Position{
					Line:      1,
					Character: 11,
				},
			},
			wantedName:          "when",
			wantedPipelineParam: false,
		},
		{
			name: "Multiple parameters in one line 1",
			args: args{
				content: []byte(`run: echo << parameters.param1 >> >> << parameters.param1 >>`),
				position: protocol.Position{
					Line:      0,
					Character: 26,
				},
			},
			wantedName:          "param1",
			wantedPipelineParam: false,
		},
		{
			name: "Multiple parameters in one line 2",
			args: args{
				content: []byte(`run: echo << parameters.param1 >> >> << parameters.param2 >>`),
				position: protocol.Position{
					Line:      0,
					Character: 42,
				},
			},
			wantedName:          "param2",
			wantedPipelineParam: false,
		},
		{
			name: "Pipeline parameters ",
			args: args{
				content: []byte(`- run:
                when: << pipeline.parameters.release >>`),
				position: protocol.Position{
					Line:      1,
					Character: 49,
				},
			},
			wantedName:          "release",
			wantedPipelineParam: true,
		},
		{
			name: "Pipeline parameters inside multiple params line",
			args: args{
				content: []byte(`run: echo << pipeline.parameters.param1 >> >> << parameters.param2 >>`),
				position: protocol.Position{
					Line:      0,
					Character: 29,
				},
			},
			wantedName:          "param1",
			wantedPipelineParam: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, isPipelineParam := NameUsedAtPos(tt.args.content, tt.args.position); got != tt.wantedName || isPipelineParam != tt.wantedPipelineParam {
				t.Errorf("NameUsedAtPos() = got %v, want %v", got, tt.wantedName)
			}
		})
	}
}

func TestCouldExpandTo(t *testing.T) {
	tests := []struct {
		content, s string
		want       bool
	}{
		{"deploy", "deploy", true},
		{"deploy", "deploy-prod", false},
		{"deploy-<< pipeline.parameters.env >>", "deploy-prod", true},
		{"deploy-<< pipeline.parameters.env >>", "deploy-", true},
		{"deploy-<< pipeline.parameters.env >>", "build-prod", false},
		{"<< pipeline.git.branch >> (<< matrix.os >>)", "main (linux)", true},
		{"a.b-<< pipeline.parameters.x >>", "aXb-1", false},
	}
	for _, tt := range tests {
		got := CouldExpandTo(tt.content, tt.s)
		assert.Check(t, cmp.Equal(got, tt.want), "CouldExpandTo(%q, %q)", tt.content, tt.s)
	}
}
