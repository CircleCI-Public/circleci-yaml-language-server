package parser

import (
	"reflect"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
)

type orbsArgs struct {
	orbsString string
}

func getOrbsTest() []struct {
	name string
	args orbsArgs
	want []ast.Orb
} {
	tests := []struct {
		name string
		args orbsArgs
		want []ast.Orb
	}{
		{
			name: "Orbs test case 1",
			args: orbsArgs{
				orbsString: `
orbs:
    superOrb: superOrb/superFunction@1.2.3
    anotherOrb: anotherOne/mehFunction@0.1.2
    localOrb:
        commands:
            some_command:
                steps:
                    - run: "echo 'hello world"`,
			},
			want: []ast.Orb{
				{
					Name: "superOrb",
					Url: ast.OrbURL{
						Name:    "superOrb/superFunction",
						Version: "1.2.3",
					},
				},
				{
					Name: "anotherOrb",
					Url: ast.OrbURL{
						Name:    "anotherOne/mehFunction",
						Version: "0.1.2",
					},
				},
			},
		},
	}
	return tests
}

func TestYamlDocument_parseSingleOrb(t *testing.T) {
	tests := getOrbsTest()

	for _, tt := range tests {
		t.Run(tt.name+": parseSingleOrb", func(t *testing.T) {
			rootNode := getNodeForString(tt.args.orbsString)
			doc := &YamlDocument{
				Content: []byte(tt.args.orbsString),
				Orbs:    make(map[string]ast.Orb),
			}
			blockMapping := GetChildOfType(rootNode, "block_mapping")
			blockMappingPair := blockMapping.Child(0)

			orb, _ := doc.parseSingleOrb(blockMappingPair)

			if !reflect.DeepEqual(tt.want[0].Name, orb.Name) {
				t.Errorf("YamlDocument.parseSingleOrb() = Name %v, want %v", tt.want[0], orb.Name)
			}
			if !reflect.DeepEqual(tt.want[0].Url, orb.Url) {
				t.Errorf("YamlDocument.parseSingleOrb() = ResourceClass %v, want %v", tt.want[0], orb.Url)
			}
		})
	}
}
