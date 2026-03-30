package parser

import (
	"reflect"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	sitter "github.com/smacker/go-tree-sitter"
)

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
