package languageservice

import (
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	utils "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestDefinition(t *testing.T) {
	cwd, _ := os.Getwd()
	schemaPath, _ := filepath.Abs(cwd + "/../../schema.json")
	os.Setenv("SCHEMA_LOCATION", schemaPath)
	cache := utils.CreateCache()

	parsedOrb, err := parser.ParseFile(uri.File(path.Join("./testdata/orb.yaml")))

	if err != nil {
		panic(err)
	}

	cache.OrbCache.SetOrb(&ast.CachedOrb{
		FilePath: uri.File(path.Join("./testdata/orb.yaml")).Filename(),
		Commands: parsedOrb.Commands,
		Jobs:     parsedOrb.Jobs,
	}, "superorb/superfunc@1.2.3")

	type args struct {
		filePath string
		position protocol.Position
	}
	tests := []struct {
		name    string
		args    args
		want    []protocol.Location
		wantErr bool
	}{
		{
			name: "Definition for job param",
			args: args{
				filePath: "./testdata/references.yml",
				position: protocol.Position{
					Line:      58,
					Character: 60,
				},
			},
			want: []protocol.Location{
				{
					URI: uri.File("./testdata/references.yml"),
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      42,
							Character: 12,
						},
						End: protocol.Position{
							Line:      43,
							Character: 28,
						},
					},
				},
			},
		},
		{
			name: "Definition for job executor",
			args: args{
				filePath: "./testdata/references.yml",
				position: protocol.Position{
					Line:      78,
					Character: 23,
				},
			},
			want: []protocol.Location{
				{
					URI: uri.File("./testdata/references.yml"),
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      28,
							Character: 4,
						},
						End: protocol.Position{
							Line:      36,
							Character: 55,
						},
					},
				},
			},
		},
		{
			name: "Definition for executor's parameter",
			args: args{
				filePath: "./testdata/references.yml",
				position: protocol.Position{
					Line:      36,
					Character: 43,
				},
			},
			want: []protocol.Location{
				{
					URI: uri.File("./testdata/references.yml"),
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      30,
							Character: 12,
						},
						End: protocol.Position{
							Line:      32,
							Character: 30,
						},
					},
				},
			},
		},
		{
			name: "Definition for job definition",
			args: args{
				filePath: "./testdata/references.yml",
				position: protocol.Position{
					Line:      73,
					Character: 13,
				},
			},
			want: []protocol.Location{
				{
					URI: uri.File("./testdata/references.yml"),
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      73,
							Character: 4,
						},
						End: protocol.Position{
							Line:      88,
							Character: 31,
						},
					},
				},
			},
		},
		{
			name: "Definition for command definition",
			args: args{
				filePath: "./testdata/references.yml",
				position: protocol.Position{
					Line:      91,
					Character: 12,
				},
			},
			want: []protocol.Location{
				{
					URI: uri.File("./testdata/references.yml"),
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      91,
							Character: 4,
						},
						End: protocol.Position{
							Line:      102,
							Character: 64,
						},
					},
				},
			},
		},
		{
			name: "Definition for command param",
			args: args{
				filePath: "./testdata/references.yml",
				position: protocol.Position{
					Line:      99,
					Character: 58,
				},
			},
			want: []protocol.Location{
				{
					URI: uri.File("./testdata/references.yml"),
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      93,
							Character: 12,
						},
						End: protocol.Position{
							Line:      94,
							Character: 28,
						},
					},
				},
			},
		},
		{
			name: "Definition for command param definition",
			args: args{
				filePath: "./testdata/references.yml",
				position: protocol.Position{
					Line:      93,
					Character: 23,
				},
			},
			want: []protocol.Location{
				{
					URI: uri.File("./testdata/references.yml"),
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      93,
							Character: 12,
						},
						End: protocol.Position{
							Line:      93,
							Character: 29,
						},
					},
				},
			},
		},
		{
			name: "Definition for command",
			args: args{
				filePath: "./testdata/references.yml",
				position: protocol.Position{
					Line:      88,
					Character: 29,
				},
			},
			want: []protocol.Location{
				{
					URI: uri.File("./testdata/references.yml"),
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      91,
							Character: 4,
						},
						End: protocol.Position{
							Line:      102,
							Character: 64,
						},
					},
				},
			},
		},
		{
			name: "Definition for job",
			args: args{
				filePath: "./testdata/references.yml",
				position: protocol.Position{
					Line:      6,
					Character: 24,
				},
			},
			want: []protocol.Location{
				{
					URI: uri.File("./testdata/references.yml"),

					Range: protocol.Range{
						Start: protocol.Position{
							Line:      39,
							Character: 4,
						},
						End: protocol.Position{
							Line:      71,
							Character: 29,
						},
					},
				},
			},
		},
		{
			name: "Definition for job ref",
			args: args{
				filePath: "./testdata/references.yml",
				position: protocol.Position{
					Line:      12,
					Character: 36,
				},
			},
			want: []protocol.Location{
				{
					URI: uri.File("./testdata/references.yml"),

					Range: protocol.Range{
						Start: protocol.Position{
							Line:      6,
							Character: 12,
						},
						End: protocol.Position{
							Line:      8,
							Character: 40,
						},
					},
				},
			},
		},
		{
			name: "Definition for orb",
			args: args{
				filePath: "./testdata/references.yml",
				position: protocol.Position{
					Line:      101,
					Character: 26,
				},
			},
			want: []protocol.Location{
				{
					URI: uri.File("./testdata/orb.yaml"),
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      9,
							Character: 4,
						},
						End: protocol.Position{
							Line:      28,
							Character: 51,
						},
					},
				},
			},
		},
		{
			name: "Definition for pipeline param",
			args: args{
				filePath: "./testdata/references.yml",
				position: protocol.Position{
					Line:      102,
					Character: 51,
				},
			},
			want: []protocol.Location{
				{
					URI: uri.File("./testdata/references.yml"),
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      110,
							Character: 4,
						},
						End: protocol.Position{
							Line:      113,
							Character: 0,
						},
					},
				},
			},
		},
		{
			name: "Definition for pipeline param definition",
			args: args{
				filePath: "./testdata/references.yml",
				position: protocol.Position{
					Line:      110,
					Character: 10,
				},
			},
			want: []protocol.Location{
				{
					URI: uri.File("./testdata/references.yml"),
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      110,
							Character: 4,
						},
						End: protocol.Position{
							Line:      110,
							Character: 14,
						},
					},
				},
			},
		},
		{
			name: "Definition for orb definition",
			args: args{
				filePath: "./testdata/references.yml",
				position: protocol.Position{
					Line:      107,
					Character: 10,
				},
			},
			want: []protocol.Location{
				{
					URI: uri.File("./testdata/orb.yaml"),
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      0,
							Character: 0,
						},
						End: protocol.Position{
							Line:      0,
							Character: 0,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, _ := os.ReadFile(tt.args.filePath)
			cache.FileCache.SetFile(&protocol.TextDocumentItem{
				URI:  uri.File(tt.args.filePath),
				Text: string(content),
			})

			params := protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: uri.File(tt.args.filePath),
					},
					Position: tt.args.position,
				},
			}

			got, err := Definition(params, cache)
			if (err != nil) != tt.wantErr {
				t.Errorf("Definition(): %s error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Definition(): %s = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
