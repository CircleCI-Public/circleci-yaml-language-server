package parser

import (
	"reflect"
	"testing"

	"github.com/circleci/circleci-yaml-language-server/pkg/ast"
	sitter "github.com/smacker/go-tree-sitter"
)

func getNodeForString(commandString string) *sitter.Node {
	rootNode := GetRootNode([]byte(commandString))
	documentNode := rootNode.Child(0)
	blockNode := GetChildOfType(documentNode, "block_node")
	blockNode = (GetChildOfType(blockNode, "block_mapping")).Child(0).ChildByFieldName("value")
	return blockNode
}

type commandArgs struct {
	commandString string
}

func getCommandsTests() []struct {
	name string
	args commandArgs
	want []ast.Command
} {
	tests := []struct {
		name string
		args commandArgs
		want []ast.Command
	}{
		{
			name: "Commands test case 1",
			args: commandArgs{
				commandString: `commands:
                someCommand:
                    description: "foo"
                    steps:
                        - checkout
                anotherCommand:
                    description: superFoo
                    steps:
                        run: "echo 'foo'"`,
			},
			want: []ast.Command{
				{
					Name:        "someCommand",
					Description: "foo",
				},
			},
		},
		{
			name: "Commands test case 2",
			args: commandArgs{
				commandString: `commands:
                someOtherCommand:
                    description: bar
                    steps:
                        - checkout`,
			},
			want: []ast.Command{
				{
					Name:        "someOtherCommand",
					Description: "bar",
				},
			},
		},
	}
	return tests
}

func TestYamlDocument_parseCommands(t *testing.T) {
	tests := getCommandsTests()

	for _, tt := range tests {

		t.Run(tt.name+": parseCommands", func(t *testing.T) {
			rootNode := getNodeForString(tt.args.commandString)
			doc := &YamlDocument{
				Content:  []byte(tt.args.commandString),
				Commands: make(map[string]ast.Command),
			}

			doc.parseCommands(rootNode)
			for _, cmd := range tt.want {
				if _, ok := doc.Commands[cmd.Name]; !ok {
					t.Errorf("YamlDocument.parseCommands() = %s could have not been found or parsed", cmd.Name)
					t.Skip()
				}
				if !reflect.DeepEqual(doc.Commands[cmd.Name].Name, cmd.Name) {
					t.Errorf("YamlDocument.parseCommands() = Name %v, want %v", doc.Commands[cmd.Name].Name, cmd.Name)
				}
				if !reflect.DeepEqual(doc.Commands[cmd.Name].Description, cmd.Description) {
					t.Errorf("YamlDocument.parseCommands() = Description %v, want %v", doc.Commands[cmd.Name].Description, cmd.Description)
				}
			}

		})
	}
}

func TestYamlDocument_parseSingleCommand(t *testing.T) {
	tests := getCommandsTests()

	for _, tt := range tests {
		t.Run(tt.name+": parseSingleCommand", func(t *testing.T) {
			rootNode := getNodeForString(tt.args.commandString)
			doc := &YamlDocument{
				Content:  []byte(tt.args.commandString),
				Commands: make(map[string]ast.Command),
			}
			blockMapping := GetChildOfType(rootNode, "block_mapping")
			blockMappingPair := blockMapping.Child(0)

			cmd := doc.parseSingleCommand(blockMappingPair)

			if !reflect.DeepEqual(cmd.Name, tt.want[0].Name) {
				t.Errorf("YamlDocument.parseSingleCommand() = Name %v, want %v", cmd.Name, tt.want[0].Name)
			}
			if !reflect.DeepEqual(cmd.Description, tt.want[0].Description) {
				t.Errorf("YamlDocument.parseSingleCommand() = Description %v, want %v", cmd.Description, tt.want[0].Description)
			}
		})
	}
}
