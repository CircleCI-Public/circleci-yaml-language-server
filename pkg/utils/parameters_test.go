package utils

import (
	"reflect"
	"testing"

	"go.lsp.dev/protocol"
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
			if got, isPipelineParam := GetParamNameUsedAtPos(tt.args.content, tt.args.position); got != tt.wantedName || isPipelineParam != tt.wantedPipelineParam {
				t.Errorf("GetParamNameUsedAtPos() = got %v, want %v", got, tt.wantedName)
			}
		})
	}
}

func TestGetParamsUsedInNode(t *testing.T) {
	type args struct {
		content string
	}

	tests := []struct {
		name string
		args args
		want []struct {
			Name       string
			FullName   string
			ParamRange protocol.Range
		}
	}{
		// Test Case 1
		{
			name: "No parameters",
			args: args{
				content: "example: somevalue",
			},
			want: []struct {
				Name       string
				FullName   string
				ParamRange protocol.Range
			}{},
		},

		// Test Case 2
		{
			name: "One parameter in oneliner",
			args: args{
				content: "example2: <<parameters.example>>",
			},
			want: []struct {
				Name       string
				FullName   string
				ParamRange protocol.Range
			}{
				{
					Name:     "example",
					FullName: "parameters.example",
					ParamRange: protocol.Range{
						Start: protocol.Position{
							Line:      0,
							Character: 10,
						},
						End: protocol.Position{
							Line:      0,
							Character: 32,
						},
					},
				},
			},
		},

		// Test Case 3
		{
			name: "Multiple parameters in oneliner",
			args: args{
				content: "example3: Hello <<parameters.name>>, welcome to <<parameters.place>>",
			},
			want: []struct {
				Name       string
				FullName   string
				ParamRange protocol.Range
			}{
				{
					Name:     "name",
					FullName: "parameters.name",
					ParamRange: protocol.Range{
						Start: protocol.Position{
							Line:      0,
							Character: 16,
						},
						End: protocol.Position{
							Line:      0,
							Character: 35,
						},
					},
				},
				{
					Name:     "place",
					FullName: "parameters.place",
					ParamRange: protocol.Range{
						Start: protocol.Position{
							Line:      0,
							Character: 48,
						},
						End: protocol.Position{
							Line:      0,
							Character: 68,
						},
					},
				},
			},
		},

		// Test Case 4
		{
			name: "One parameter in block scalar",
			args: args{
				content: `
example4: |
  param on first line <<parameters.inblock>>`,
			},
			want: []struct {
				Name       string
				FullName   string
				ParamRange protocol.Range
			}{
				{
					Name:     "inblock",
					FullName: "parameters.inblock",
					ParamRange: protocol.Range{
						Start: protocol.Position{
							Line:      2,
							Character: 22,
						},
						End: protocol.Position{
							Line:      2,
							Character: 44,
						},
					},
				},
			},
		},

		// Test Case 5
		{
			name: "Multiple parameters in block scalar",
			args: args{
				content: `
example5: |
  param on first line <<parameters.inblock>> <<parameters.anotherone>>
      just some
    random string
   w i th a BAD <<// <<parameters.structure>>
       <<pipeline.parameters.lastone>>`,
			},
			want: []struct {
				Name       string
				FullName   string
				ParamRange protocol.Range
			}{
				{
					Name:     "inblock",
					FullName: "parameters.inblock",
					ParamRange: protocol.Range{
						Start: protocol.Position{
							Line:      2,
							Character: 22,
						},
						End: protocol.Position{
							Line:      2,
							Character: 44,
						},
					},
				},
				{
					Name:     "anotherone",
					FullName: "parameters.anotherone",
					ParamRange: protocol.Range{
						Start: protocol.Position{
							Line:      2,
							Character: 45,
						},
						End: protocol.Position{
							Line:      2,
							Character: 70,
						},
					},
				},
				{
					Name:     "structure",
					FullName: "parameters.structure",
					ParamRange: protocol.Range{
						Start: protocol.Position{
							Line:      5,
							Character: 21,
						},
						End: protocol.Position{
							Line:      5,
							Character: 45,
						},
					},
				},
				{
					Name:     "lastone",
					FullName: "pipeline.parameters.lastone",
					ParamRange: protocol.Range{
						Start: protocol.Position{
							Line:      6,
							Character: 7,
						},
						End: protocol.Position{
							Line:      6,
							Character: 38,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := GetParamsInString(tt.args.content)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetParamsUsedInNode() = %v, want %v", got, tt.want)
			}
		})
	}
}
