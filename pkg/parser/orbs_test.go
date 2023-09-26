package parser

import (
	"reflect"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
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

func createOrbPart(text string, line, start, end uint32) ast.TextAndRange {
	return ast.TextAndRange{
		Text: text,
		Range: protocol.Range{
			Start: protocol.Position{Line: line, Character: start},
			End:   protocol.Position{Line: line, Character: end},
		},
	}
}

func TestParseOrbDefinition(t *testing.T) {
	type TestCase struct {
		Name     string
		Text     string
		Range    protocol.Range
		Expected ast.OrbURLDefinition
	}

	testCases := []TestCase{
		{
			Name: "Basic behavior",
			Text: "a/b@c",
			Range: protocol.Range{
				Start: protocol.Position{0, 0},
				End:   protocol.Position{0, 5},
			},
			Expected: ast.OrbURLDefinition{
				Namespace: createOrbPart("a", 0, 0, 1),
				Name:      createOrbPart("b", 0, 2, 3),
				Version:   createOrbPart("c", 0, 4, 5),
			},
		},
		{
			Name: "Basic behavior with strange range",
			Text: "a/b@c",
			Range: protocol.Range{
				Start: protocol.Position{113, 42},
				End:   protocol.Position{113, 47},
			},
			Expected: ast.OrbURLDefinition{
				Namespace: createOrbPart("a", 113, 42, 43),
				Name:      createOrbPart("b", 113, 44, 45),
				Version:   createOrbPart("c", 113, 46, 47),
			},
		},
		{
			Name: "Stop for no version",
			Text: "a/b",
			Range: protocol.Range{
				Start: protocol.Position{0, 0},
				End:   protocol.Position{0, 3},
			},
			Expected: ast.OrbURLDefinition{
				Namespace: createOrbPart("a", 0, 0, 1),
				Name:      createOrbPart("b", 0, 2, 3),
			},
		},
		{
			Name: "Stop for no name",
			Text: "a",
			Range: protocol.Range{
				Start: protocol.Position{0, 0},
				End:   protocol.Position{0, 1},
			},
			Expected: ast.OrbURLDefinition{
				Namespace: createOrbPart("a", 0, 0, 1),
			},
		},
		{
			Name: "Detect start of name",
			Text: "a/",
			Range: protocol.Range{
				Start: protocol.Position{0, 0},
				End:   protocol.Position{0, 2},
			},
			Expected: ast.OrbURLDefinition{
				Namespace: createOrbPart("a", 0, 0, 1),
				Name:      createOrbPart("", 0, 2, 2),
			},
		},
		{
			Name: "Detect start of version",
			Text: "a/b@",
			Range: protocol.Range{
				Start: protocol.Position{0, 0},
				End:   protocol.Position{0, 4},
			},
			Expected: ast.OrbURLDefinition{
				Namespace: createOrbPart("a", 0, 0, 1),
				Name:      createOrbPart("b", 0, 2, 3),
				Version:   createOrbPart("", 0, 4, 4),
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			def := getOrbDefinitionFromTextAndRange(testCase.Text, testCase.Range)
			assert.Equal(t, testCase.Expected, def)
		})
	}
}
