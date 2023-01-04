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

func TestYamlDocument_parseOrbs(t *testing.T) {
	tests := getOrbsTest()

	for _, tt := range tests {
		t.Run(tt.name+": parseOrbs", func(t *testing.T) {
			doc := &YamlDocument{
				Commands: make(map[string]ast.Command),
				Content:  []byte(tt.args.orbsString),
				Orbs:     make(map[string]ast.Orb),

				LocalOrbInfo: make(map[string]*ast.OrbInfo),
			}
			orbNode := getNodeForString(tt.args.orbsString)

			doc.parseOrbs(orbNode)

			for _, orb := range tt.want {
				if _, ok := doc.Orbs[orb.Name]; !ok {
					t.Errorf("YamlDocument.parseOrbs() = %s could have not been found or parsed", orb.Name)
					t.Skip()
				}

				if !reflect.DeepEqual(doc.Orbs[orb.Name].Name, orb.Name) {
					t.Errorf("YamlDocument.parseOrbs() = Name %v, want %v", doc.Orbs[orb.Name], orb.Name)
				}
				if !reflect.DeepEqual(doc.Orbs[orb.Name].Url.Name, orb.Url.Name) {
					t.Errorf("YamlDocument.parseOrbs() = Url %v, want %v", doc.Orbs[orb.Name].Url.Name, orb.Url.Name)
				}
				if !reflect.DeepEqual(doc.Orbs[orb.Name].Url.Version, orb.Url.Version) {
					t.Errorf("YamlDocument.parseOrbs() = Url %v, want %v", doc.Orbs[orb.Name].Url.Version, orb.Url.Version)
				}
			}

			if _, ok := doc.Commands["localOrb/some_command"]; !ok {
				t.Error("Orb command is not declared")
			}

		})
	}
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
