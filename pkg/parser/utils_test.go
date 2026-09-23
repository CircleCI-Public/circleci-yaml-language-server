package parser

import (
	"reflect"
	"testing"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

const YamlFloat = "foo"
const YamlSeq = "- abc\n- xyz\n- def"
const YamlMap = `foo:
  abc: 1`
const YamlBig = `foo:
  abc:
    - 1
    - 2`

var FloatNode = permanentRootOf([]byte(YamlFloat)).Child(0).Child(0)
var SeqNode = permanentRootOf([]byte(YamlSeq)).Child(0).Child(0)
var MapNode = permanentRootOf([]byte(YamlMap)).Child(0).Child(0)
var BigNode = permanentRootOf([]byte(YamlBig)).Child(0).Child(0)

func TestGetChildOfType(t *testing.T) {

	tests := []struct {
		name     string
		yamlNode *sitter.Node
		typeName string
	}{
		{
			name:     "Get block_mapping",
			yamlNode: MapNode,
			typeName: "block_mapping",
		},
		{
			name:     "Get plain_scalar",
			yamlNode: FloatNode,
			typeName: "plain_scalar",
		},
		{
			name:     "Get block_sequence",
			yamlNode: SeqNode,
			typeName: "block_sequence",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetChildOfType(tt.yamlNode, tt.typeName); !reflect.DeepEqual(got.Kind(), tt.typeName) {
				t.Errorf("GetChildOfType() = %v, want %v", got.Kind(), tt.typeName)
			}
		})
	}

	// nil cases
	t.Run("Nil cases", func(t *testing.T) {
		if got := GetChildOfType(SeqNode, "flow_node"); got != nil {
			t.Errorf("GetChildOfType() = %v, want %v", &got, nil)
		}
	})
}

func Test_getBlockMappingNode(t *testing.T) {

	tests := []struct {
		name       string
		streamNode *sitter.Node
		want       string
	}{
		{
			name:       "Succeeding test case",
			streamNode: rootNodeOf(t, []byte(YamlBig)),
			want:       "block_mapping",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetBlockMappingNode(tt.streamNode); !reflect.DeepEqual(got.Kind(), tt.want) {
				t.Errorf("getBlockMappingNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

// getFirstChildOfType finds the shallowest node of a type, breadth first.
func getFirstChildOfType(rootNode *sitter.Node, typeName string) *sitter.Node {
	queue := []*sitter.Node{rootNode}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		if node.Kind() == typeName {
			return node
		}
		for i := uint(0); i < node.ChildCount(); i++ {
			queue = append(queue, node.Child(i))
		}
	}
	return nil
}

func TestGetFirstChildOfType(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		typeName string
		wantNil  bool
	}{
		{
			name:     "Find nested block_mapping_pair",
			yaml:     YamlMap,
			typeName: "block_mapping_pair",
		},
		{
			name:     "Find block_sequence in nested structure",
			yaml:     YamlBig,
			typeName: "block_sequence",
		},
		{
			name:     "Return nil for non-existent type",
			yaml:     YamlFloat,
			typeName: "flow_sequence",
			wantNil:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getFirstChildOfType(rootNodeOf(t, []byte(tt.yaml)), tt.typeName)
			if tt.wantNil {
				if got != nil {
					t.Errorf("getFirstChildOfType() = %v, want nil", got.Kind())
				}
				return
			}
			if got == nil {
				t.Fatal("getFirstChildOfType() = nil, want non-nil")
			}
			if got.Kind() != tt.typeName {
				t.Errorf("getFirstChildOfType() type = %v, want %v", got.Kind(), tt.typeName)
			}
		})
	}
}

func TestYamlDocument_GetNodeText(t *testing.T) {
	type fields struct {
		Content []byte
	}
	type args struct {
		node *sitter.Node
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name:   "Full text 1",
			fields: fields{Content: []byte(YamlFloat)},
			args:   args{node: FloatNode},
			want:   YamlFloat,
		},
		{
			name:   "Full text 2",
			fields: fields{Content: []byte(YamlSeq)},
			args:   args{node: SeqNode},
			want:   YamlSeq,
		},
		{
			name:   "Full text 3",
			fields: fields{Content: []byte(YamlMap)},
			args:   args{node: MapNode},
			want:   YamlMap,
		},
		{
			name:   "Subnode Text",
			fields: fields{Content: []byte(YamlMap)},
			args: args{
				node: GetChildOfType(GetChildOfType(MapNode, "block_mapping"), "block_mapping_pair").ChildByFieldName("key"),
			},
			want: "foo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &YamlDocument{
				Content: tt.fields.Content,
			}
			if got := doc.GetNodeText(tt.args.node); got != tt.want {
				t.Errorf("YamlDocument.GetNodeText() = %v, want %v", got, tt.want)
			}
		})
	}
}
