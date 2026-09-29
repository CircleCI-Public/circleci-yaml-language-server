package paramref

import (
	"strings"
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

func TestCouldBothExpandTo(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"setup-<< pipeline.parameters.env >>", "setup-<< pipeline.parameters.env >>", true},
		{"setup-<< pipeline.parameters.env >>", "setup-<< pipeline.parameters.other >>", true},
		{"<< pipeline.parameters.env >>-setup", "setup-<< pipeline.parameters.env >>", true},
		{"test-hello-<< pipeline.parameters.place >>", "test-<< pipeline.parameters.x >>", true},
		{"test-<< pipeline.parameters.x >>", "deploy-<< pipeline.parameters.x >>", false},
		{"<< pipeline.parameters.x >>-a", "<< pipeline.parameters.x >>-b", false},
	}
	for _, tt := range tests {
		got := CouldBothExpandTo(tt.a, tt.b)
		assert.Check(t, cmp.Equal(got, tt.want), "CouldBothExpandTo(%q, %q)", tt.a, tt.b)
	}
}

func TestSections(t *testing.T) {
	const content = `run: echo <<# parameters.loud >>-v<</ parameters.loud >> <<^ pipeline.parameters.quiet >>-n<</ pipeline.parameters.quiet >>`
	// at is the end of text, on the name of the parameter it ends with.
	at := func(text string) protocol.Position {
		return protocol.Position{Character: uint32(strings.Index(content, text) + len(text))}
	}

	t.Run("a section's opening tag names its parameter", func(t *testing.T) {
		name, isPipelineParam := NameUsedAtPos([]byte(content), at("<<# parameters.loud"))
		assert.Check(t, cmp.Equal(name, "loud"))
		assert.Check(t, !isPipelineParam)
	})

	t.Run("a section's closing tag names its parameter", func(t *testing.T) {
		name, isPipelineParam := NameUsedAtPos([]byte(content), at("<</ parameters.loud"))
		assert.Check(t, cmp.Equal(name, "loud"))
		assert.Check(t, !isPipelineParam)
	})

	t.Run("an inverted section names a pipeline parameter", func(t *testing.T) {
		name, isPipelineParam := NameUsedAtPos([]byte(content), at("<<^ pipeline.parameters.quiet"))
		assert.Check(t, cmp.Equal(name, "quiet"))
		assert.Check(t, isPipelineParam)
	})

	t.Run("both of a section's tags are references", func(t *testing.T) {
		whole := protocol.Range{End: protocol.Position{Character: uint32(len(content))}}
		got, err := ReferencesInRange([]byte(content), "loud", whole)
		assert.NilError(t, err)
		texts := []string{}
		for _, reference := range got {
			texts = append(texts, content[reference[0]:reference[1]])
		}
		assert.Check(t, cmp.DeepEqual(texts, []string{"<<# parameters.loud >>", "<</ parameters.loud >>"}))
	})
}
