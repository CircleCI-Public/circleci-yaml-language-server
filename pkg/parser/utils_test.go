package parser

import (
	"reflect"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

const YamlFloat = "foo"
const YamlSeq = "- abc\n- xyz\n- def"
const YamlMap = `foo:
  abc: 1`
const YamlBig = `foo:
  abc:
    - 1
    - 2`

var FloatNode = GetRootNode([]byte(YamlFloat)).Child(0).Child(0)
var SeqNode = GetRootNode([]byte(YamlSeq)).Child(0).Child(0)
var MapNode = GetRootNode([]byte(YamlMap)).Child(0).Child(0)
var BigNode = GetRootNode([]byte(YamlBig)).Child(0).Child(0)

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
			if got := GetChildOfType(tt.yamlNode, tt.typeName); !reflect.DeepEqual(got.Type(), tt.typeName) {
				t.Errorf("GetChildOfType() = %v, want %v", got.Type(), tt.typeName)
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
			streamNode: GetRootNode([]byte(YamlBig)),
			want:       "block_mapping",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetBlockMappingNode(tt.streamNode); !reflect.DeepEqual(got.Type(), tt.want) {
				t.Errorf("getBlockMappingNode() = %v, want %v", got, tt.want)
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
