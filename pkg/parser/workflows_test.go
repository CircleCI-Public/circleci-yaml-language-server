package parser

import (
	"reflect"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

func getFirstChildOfType(rootNode *sitter.Node, typeName string) *sitter.Node {
	iter := sitter.NewIterator(rootNode, sitter.BFSMode)
	node, err := iter.Next()
	for err == nil {
		if node.Type() == typeName {
			return node
		}
		node, err = iter.Next()
	}
	return nil
}

func TestYamlDocument_parseSingleJobInvocation(t *testing.T) {
	const jobInvocation1 = "- build"
	const jobInvocation2 = `
- test:
    requires:
        - setup`
	const jobInvocation3 = `
- test:
    matrix:
        parameters:
            bar: [1, 2]`
	const jobInvocation4 = `
- test:
    name: say-my-name`
	const jobInvocation5 = `
- test:
    requires:
        - setup: failed`
	const jobInvocation6 = `
- test:
    requires:
        - setup: [success, canceled]`
	const jobInvocation7 = `
- deploy:
    serial-group: deploy-group`
	const jobInvocation8 = `
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
			fields: fields{Content: []byte(jobInvocation1)},
			args:   args{jobInvocationNode: getFirstChildOfType(GetRootNode([]byte(jobInvocation1)), "block_sequence_item")},
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
			fields: fields{Content: []byte(jobInvocation2)},
			args:   args{jobInvocationNode: getFirstChildOfType(GetRootNode([]byte(jobInvocation2)), "block_sequence_item")},
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
			fields: fields{Content: []byte(jobInvocation3)},
			args:   args{jobInvocationNode: getFirstChildOfType(GetRootNode([]byte(jobInvocation3)), "block_sequence_item")},
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
			name:   "Named job invocation with matrix parameters",
			fields: fields{Content: []byte(jobInvocation4)},
			args:   args{jobInvocationNode: getFirstChildOfType(GetRootNode([]byte(jobInvocation4)), "block_sequence_item")},
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
			fields: fields{Content: []byte(jobInvocation5)},
			args:   args{jobInvocationNode: getFirstChildOfType(GetRootNode([]byte(jobInvocation5)), "block_sequence_item")},
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
			fields: fields{Content: []byte(jobInvocation6)},
			args:   args{jobInvocationNode: getFirstChildOfType(GetRootNode([]byte(jobInvocation6)), "block_sequence_item")},
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
			fields: fields{Content: []byte(jobInvocation7)},
			args:   args{jobInvocationNode: getFirstChildOfType(GetRootNode([]byte(jobInvocation7)), "block_sequence_item")},
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
			fields: fields{Content: []byte(jobInvocation8)},
			args:   args{jobInvocationNode: getFirstChildOfType(GetRootNode([]byte(jobInvocation8)), "block_sequence_item")},
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

func TestYamlDocument_parseWorkflows(t *testing.T) {
	const worfklows1 = `
    test-build:
        jobs:
          - build`

	var workflowsNode1 = getFirstChildOfType(GetRootNode([]byte(worfklows1)), "block_node")
	type fields struct {
		Content   []byte
		Workflows map[string]ast.Workflow
	}
	type args struct {
		workflowsNode *sitter.Node
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string]ast.Workflow
	}{
		{
			name:   "Simple worflow",
			fields: fields{Content: []byte(worfklows1), Workflows: make(map[string]ast.Workflow)},
			args:   args{workflowsNode: workflowsNode1},
			want: map[string]ast.Workflow{
				"test-build": {Name: "test-build", JobInvocations: []ast.JobInvocation{{JobName: "build"}}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &YamlDocument{
				Content:   tt.fields.Content,
				Workflows: tt.fields.Workflows,
			}
			doc.parseWorkflows(tt.args.workflowsNode)
			parsedWorflows := doc.Workflows

			for _, wf := range tt.want {
				if _, ok := parsedWorflows[wf.Name]; !ok {
					t.Errorf("YamlDocument.parseWorkflows() did not parse workflow %s", wf.Name)
					t.Skip()
				}

				if !reflect.DeepEqual(parsedWorflows[wf.Name].Name, wf.Name) {
					t.Errorf("YamlDocument.parseWorkflows().Name = %v, want %v", parsedWorflows[wf.Name].Name, wf.Name)
				}
			}

		})
	}
}
