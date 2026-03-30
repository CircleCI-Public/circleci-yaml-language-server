package parser

import (
	"reflect"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

func TestYamlDocument_parseSingleJobInvocation(t *testing.T) {
	const jobInv1 = "- build"
	const jobInv2 = `
- test:
    requires:
        - setup`
	const jobInv3 = `
- test:
    matrix:
        parameters:
            bar: [1, 2]`
	const jobInv4 = `
- test:
    name: say-my-name`
	const jobInv5 = `
- test:
    requires:
        - setup: failed`
	const jobInv6 = `
- test:
    requires:
        - setup: [success, canceled]`
	const jobInv7 = `
- deploy:
    serial-group: deploy-group`
	const jobInv8 = `
- deploy:
    override-with: foo/deploy`

	type fields struct {
		Content   []byte
		Workflows map[string]ast.Workflow
	}
	type args struct {
		jobInvocationNode *sitter.Node
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   ast.JobInvocation
	}{
		{
			name:   "Simple named job invocation",
			fields: fields{Content: []byte(jobInv1)},
			args:   args{jobInvocationNode: getFirstChildOfType(GetRootNode([]byte(jobInv1)), "block_sequence_item")},
			want: ast.JobInvocation{
				JobName: "build",
				JobInvocationRange: protocol.Range{
					Start: protocol.Position{
						Line:      0,
						Character: 0,
					},
					End: protocol.Position{
						Line:      0,
						Character: 7,
					},
				},
				JobNameRange: protocol.Range{
					Start: protocol.Position{
						Line:      0,
						Character: 2,
					},
					End: protocol.Position{
						Line:      0,
						Character: 7,
					},
				},
				StepName: "build",
				StepNameRange: protocol.Range{
					Start: protocol.Position{
						Line:      0,
						Character: 2,
					},
					End: protocol.Position{
						Line:      0,
						Character: 7,
					},
				},
				Parameters:   make(map[string]ast.ParameterValue),
				HasMatrix:    false,
				MatrixParams: make(map[string][]ast.ParameterValue),
			},
		},
		{
			name:   "Named job invocation with parameters",
			fields: fields{Content: []byte(jobInv2)},
			args:   args{jobInvocationNode: getFirstChildOfType(GetRootNode([]byte(jobInv2)), "block_sequence_item")},
			want: ast.JobInvocation{
				JobName: "test",
				JobInvocationRange: protocol.Range{
					Start: protocol.Position{
						Line:      1,
						Character: 0,
					},
					End: protocol.Position{
						Line:      3,
						Character: 15,
					},
				},
				JobNameRange: protocol.Range{
					Start: protocol.Position{
						Line:      1,
						Character: 2,
					},
					End: protocol.Position{
						Line:      1,
						Character: 6,
					},
				},
				StepName: "test",
				StepNameRange: protocol.Range{
					Start: protocol.Position{
						Line:      1,
						Character: 2,
					},
					End: protocol.Position{
						Line:      1,
						Character: 6,
					},
				},
				Requires: []ast.Require{
					{
						Name:   "setup",
						Status: []string{"success"},
						Range: protocol.Range{
							Start: protocol.Position{Line: 3, Character: 10},
							End:   protocol.Position{Line: 3, Character: 15},
						},
						StatusRange: protocol.Range{
							Start: protocol.Position{Line: 3, Character: 10},
							End:   protocol.Position{Line: 3, Character: 15},
						},
					},
				},
				MatrixParams: make(map[string][]ast.ParameterValue),
				Parameters:   make(map[string]ast.ParameterValue),
			},
		},
		{
			name:   "Named job invocation with matrix parameters",
			fields: fields{Content: []byte(jobInv3)},
			args:   args{jobInvocationNode: getFirstChildOfType(GetRootNode([]byte(jobInv3)), "block_sequence_item")},
			want: ast.JobInvocation{
				JobName: "test",
				JobInvocationRange: protocol.Range{
					Start: protocol.Position{
						Line:      1,
						Character: 0,
					},
					End: protocol.Position{
						Line:      4,
						Character: 23,
					},
				},
				JobNameRange: protocol.Range{
					Start: protocol.Position{
						Line:      1,
						Character: 2,
					},
					End: protocol.Position{
						Line:      1,
						Character: 6,
					},
				},
				StepName: "test",
				StepNameRange: protocol.Range{
					Start: protocol.Position{
						Line:      1,
						Character: 2,
					},
					End: protocol.Position{
						Line:      1,
						Character: 6,
					},
				},
				HasMatrix: true,
				MatrixRange: protocol.Range{
					Start: protocol.Position{
						Line:      2,
						Character: 4,
					},
					End: protocol.Position{
						Line:      4,
						Character: 23,
					},
				},
				MatrixParams: map[string][]ast.ParameterValue{
					"bar": {
						{
							Name: "bar",
							Type: "enum",
							Value: []ast.ParameterValue{
								{
									Name:  "bar",
									Value: 1,
									Type:  "integer",
									ValueRange: protocol.Range{
										Start: protocol.Position{
											Line:      4,
											Character: 18,
										},
										End: protocol.Position{
											Line:      4,
											Character: 19,
										},
									},
									Range: protocol.Range{
										Start: protocol.Position{
											Line:      4,
											Character: 18,
										},
										End: protocol.Position{
											Line:      4,
											Character: 19,
										},
									},
								},
								{
									Name:  "bar",
									Value: 2,
									Type:  "integer",
									ValueRange: protocol.Range{
										Start: protocol.Position{
											Line:      4,
											Character: 21,
										},
										End: protocol.Position{
											Line:      4,
											Character: 22,
										},
									},
									Range: protocol.Range{
										Start: protocol.Position{
											Line:      4,
											Character: 21,
										},
										End: protocol.Position{
											Line:      4,
											Character: 22,
										},
									},
								},
							},
							ValueRange: protocol.Range{
								Start: protocol.Position{
									Line:      4,
									Character: 17,
								},
								End: protocol.Position{
									Line:      4,
									Character: 23,
								},
							},
							Range: protocol.Range{
								Start: protocol.Position{
									Line:      4,
									Character: 12,
								},
								End: protocol.Position{
									Line:      4,
									Character: 23,
								},
							},
						},
					},
				},
				Parameters: make(map[string]ast.ParameterValue),
			},
		},
		{
			name:   "Named job invocation with explicit step name",
			fields: fields{Content: []byte(jobInv4)},
			args:   args{jobInvocationNode: getFirstChildOfType(GetRootNode([]byte(jobInv4)), "block_sequence_item")},
			want: ast.JobInvocation{
				JobName: "test",
				JobNameRange: protocol.Range{
					Start: protocol.Position{
						Line:      1,
						Character: 2,
					},
					End: protocol.Position{
						Line:      1,
						Character: 6,
					},
				},
				StepName: "say-my-name",
				StepNameRange: protocol.Range{
					Start: protocol.Position{
						Line:      2,
						Character: 10,
					},
					End: protocol.Position{
						Line:      2,
						Character: 21,
					},
				},
				JobInvocationRange: protocol.Range{
					Start: protocol.Position{
						Line:      1,
						Character: 0,
					},
					End: protocol.Position{
						Line:      2,
						Character: 21,
					},
				},
				Parameters:   make(map[string]ast.ParameterValue),
				HasMatrix:    false,
				MatrixParams: make(map[string][]ast.ParameterValue),
			},
		},
		{
			name:   "Job invocation with requires and single status",
			fields: fields{Content: []byte(jobInv5)},
			args:   args{jobInvocationNode: getFirstChildOfType(GetRootNode([]byte(jobInv5)), "block_sequence_item")},
			want: ast.JobInvocation{
				JobName: "test",
				JobInvocationRange: protocol.Range{
					Start: protocol.Position{
						Line:      1,
						Character: 0,
					},
					End: protocol.Position{
						Line:      3,
						Character: 23,
					},
				},
				JobNameRange: protocol.Range{
					Start: protocol.Position{
						Line:      1,
						Character: 2,
					},
					End: protocol.Position{
						Line:      1,
						Character: 6,
					},
				},
				StepName: "test",
				StepNameRange: protocol.Range{
					Start: protocol.Position{
						Line:      1,
						Character: 2,
					},
					End: protocol.Position{
						Line:      1,
						Character: 6,
					},
				},
				Requires: []ast.Require{
					{
						Name:   "setup",
						Status: []string{"failed"},
						Range: protocol.Range{
							Start: protocol.Position{Line: 3, Character: 10},
							End:   protocol.Position{Line: 3, Character: 15},
						},
						StatusRange: protocol.Range{
							Start: protocol.Position{Line: 3, Character: 17},
							End:   protocol.Position{Line: 3, Character: 23},
						},
					},
				},
				MatrixParams: make(map[string][]ast.ParameterValue),
				Parameters:   make(map[string]ast.ParameterValue),
			},
		},
		{
			name:   "Job invocation with requires and multiple statuses",
			fields: fields{Content: []byte(jobInv6)},
			args:   args{jobInvocationNode: getFirstChildOfType(GetRootNode([]byte(jobInv6)), "block_sequence_item")},
			want: ast.JobInvocation{
				JobName: "test",
				JobInvocationRange: protocol.Range{
					Start: protocol.Position{
						Line:      1,
						Character: 0,
					},
					End: protocol.Position{
						Line:      3,
						Character: 36,
					},
				},
				JobNameRange: protocol.Range{
					Start: protocol.Position{
						Line:      1,
						Character: 2,
					},
					End: protocol.Position{
						Line:      1,
						Character: 6,
					},
				},
				StepName: "test",
				StepNameRange: protocol.Range{
					Start: protocol.Position{
						Line:      1,
						Character: 2,
					},
					End: protocol.Position{
						Line:      1,
						Character: 6,
					},
				},
				Requires: []ast.Require{
					{
						Name:   "setup",
						Status: []string{"success", "canceled"},
						Range: protocol.Range{
							Start: protocol.Position{Line: 3, Character: 10},
							End:   protocol.Position{Line: 3, Character: 15},
						},
						StatusRange: protocol.Range{
							Start: protocol.Position{Line: 3, Character: 17},
							End:   protocol.Position{Line: 3, Character: 36},
						},
					},
				},
				MatrixParams: make(map[string][]ast.ParameterValue),
				Parameters:   make(map[string]ast.ParameterValue),
			},
		},
		{
			name:   "Job invocation with serial group",
			fields: fields{Content: []byte(jobInv7)},
			args:   args{jobInvocationNode: getFirstChildOfType(GetRootNode([]byte(jobInv7)), "block_sequence_item")},
			want: ast.JobInvocation{
				JobName: "deploy",
				JobInvocationRange: protocol.Range{
					Start: protocol.Position{
						Line:      1,
						Character: 0,
					},
					End: protocol.Position{
						Line:      2,
						Character: 30,
					},
				},
				JobNameRange: protocol.Range{
					Start: protocol.Position{
						Line:      1,
						Character: 2,
					},
					End: protocol.Position{
						Line:      1,
						Character: 8,
					},
				},
				StepName: "deploy",
				StepNameRange: protocol.Range{
					Start: protocol.Position{
						Line:      1,
						Character: 2,
					},
					End: protocol.Position{
						Line:      1,
						Character: 8,
					},
				},
				Parameters:   make(map[string]ast.ParameterValue),
				HasMatrix:    false,
				MatrixParams: make(map[string][]ast.ParameterValue),
				SerialGroup:  "deploy-group",
				SerialGroupRange: protocol.Range{
					Start: protocol.Position{
						Line:      2,
						Character: 18,
					},
					End: protocol.Position{
						Line:      2,
						Character: 30,
					},
				},
			},
		},
		{
			name:   "Job invocation with override",
			fields: fields{Content: []byte(jobInv8)},
			args:   args{jobInvocationNode: getFirstChildOfType(GetRootNode([]byte(jobInv8)), "block_sequence_item")},
			want: ast.JobInvocation{
				JobName: "deploy",
				JobInvocationRange: protocol.Range{
					Start: protocol.Position{
						Line:      1,
						Character: 0,
					},
					End: protocol.Position{
						Line:      2,
						Character: 29,
					},
				},
				JobNameRange: protocol.Range{
					Start: protocol.Position{
						Line:      1,
						Character: 2,
					},
					End: protocol.Position{
						Line:      1,
						Character: 8,
					},
				},
				StepName: "deploy",
				StepNameRange: protocol.Range{
					Start: protocol.Position{
						Line:      1,
						Character: 2,
					},
					End: protocol.Position{
						Line:      1,
						Character: 8,
					},
				},
				Parameters:   make(map[string]ast.ParameterValue),
				HasMatrix:    false,
				MatrixParams: make(map[string][]ast.ParameterValue),
				OverrideWith: "foo/deploy",
				OverrideWithRange: protocol.Range{
					Start: protocol.Position{
						Line:      2,
						Character: 19,
					},
					End: protocol.Position{
						Line:      2,
						Character: 29,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &YamlDocument{
				Content: tt.fields.Content,
			}
			got := doc.parseSingleJobInvocation(tt.args.jobInvocationNode)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("YamlDocument.parseSingleJobInvocation() = got %v, want %v", got, tt.want)
			}
		})
	}
}
