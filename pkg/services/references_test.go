package languageservice

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/testHelpers"
	utils "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestReferences(t *testing.T) {
	cwd, _ := os.Getwd()
	schemaPath, _ := filepath.Abs(cwd + "/../../schema.json")
	os.Setenv("SCHEMA_LOCATION", schemaPath)
	cache := utils.CreateCache()

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
			name: "Reference for job param",
			args: args{
				filePath: "./testdata/references.yml",
				position: protocol.Position{
					Line:      42,
					Character: 22,
				},
			},
			want: []protocol.Location{
				{
					URI: uri.File("./testdata/references.yml"),
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      58,
							Character: 35,
						},
						End: protocol.Position{
							Line:      58,
							Character: 69,
						},
					},
				},
				{
					URI: uri.File("./testdata/references.yml"),
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      66,
							Character: 35,
						},
						End: protocol.Position{
							Line:      66,
							Character: 69,
						},
					},
				},
			},
		},
		{
			name: "Reference for command",
			args: args{
				filePath: "./testdata/references.yml",
				position: protocol.Position{
					Line:      91,
					Character: 17,
				},
			},
			want: []protocol.Location{
				{
					URI: uri.File("./testdata/references.yml"),
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      88,
							Character: 14,
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
			name: "Reference for job",
			args: args{
				filePath: "./testdata/references.yml",
				position: protocol.Position{
					Line:      39,
					Character: 22,
				},
			},
			want: []protocol.Location{
				{
					URI: uri.File("./testdata/references.yml"),

					Range: protocol.Range{
						Start: protocol.Position{
							Line:      6,
							Character: 14,
						},
						End: protocol.Position{
							Line:      6,
							Character: 33,
						},
					},
				},
			},
		},
		{
			name: "Reference for orb",
			args: args{
				filePath: "./testdata/references.yml",
				position: protocol.Position{
					Line:      107,
					Character: 10,
				},
			},
			want: []protocol.Location{
				{
					URI: uri.File("./testdata/references.yml"),
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      101,
							Character: 14,
						},
						End: protocol.Position{
							Line:      101,
							Character: 34,
						},
					},
				},
			},
		},
		{
			name: "Reference for workflow",
			args: args{
				filePath: "./testdata/references.yml",
				position: protocol.Position{
					Line:      17,
					Character: 23,
				},
			},
			want: []protocol.Location{
				{
					URI: uri.File("./testdata/references.yml"),
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      17,
							Character: 14,
						},
						End: protocol.Position{
							Line:      17,
							Character: 29,
						},
					},
				},
			},
		},
		{
			name: "Reference for pipeline param",
			args: args{
				filePath: "./testdata/references.yml",
				position: protocol.Position{
					Line:      110,
					Character: 9,
				},
			},
			want: []protocol.Location{
				{
					URI: uri.File("./testdata/references.yml"),
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      102,
							Character: 28,
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
			name: "Reference for executor",
			args: args{
				filePath: "./testdata/references.yml",
				position: protocol.Position{
					Line:      28,
					Character: 9,
				},
			},
			want: []protocol.Location{
				{
					URI: uri.File("./testdata/references.yml"),
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      50,
							Character: 8,
						},
						End: protocol.Position{
							Line:      50,
							Character: 25,
						},
					},
				},
				{
					URI: uri.File("./testdata/references.yml"),
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      78,
							Character: 8,
						},
						End: protocol.Position{
							Line:      78,
							Character: 25,
						},
					},
				},
			},
		},
		{
			name: "Reference for executor's parameter",
			args: args{
				filePath: "./testdata/references.yml",
				position: protocol.Position{
					Line:      30,
					Character: 18,
				},
			},
			want: []protocol.Location{
				{
					URI: uri.File("./testdata/references.yml"),
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      36,
							Character: 24,
						},
						End: protocol.Position{
							Line:      36,
							Character: 55,
						},
					},
				},
			},
		},
	}
	context := testHelpers.GetDefaultLsContext()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, _ := os.ReadFile(tt.args.filePath)
			cache.FileCache.SetFile(utils.CachedFile{
				TextDocument: protocol.TextDocumentItem{
					URI:  uri.File(tt.args.filePath),
					Text: string(content),
				},
				Project:      utils.Project{},
				EnvVariables: make([]string, 0),
			})

			params := protocol.ReferenceParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: uri.File(tt.args.filePath),
					},
					Position: tt.args.position,
				},
			}

			got, err := References(params, cache, context)

			// We don't care about the order of the items,
			// so we sort them before comparing to avoid the order
			// being the reason the test doesn't pass.
			sortLocationItem(got)
			sortLocationItem(tt.want)
			if (err != nil) != tt.wantErr {
				t.Errorf("References(): %s error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("References(): %s = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func sortLocationItem(items []protocol.Location) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Range.Start.Line == items[j].Range.Start.Line {
			return items[i].Range.Start.Character < items[j].Range.Start.Character
		}
		if items[i].Range.End.Line == items[j].Range.End.Line {
			return items[i].Range.End.Character < items[j].Range.End.Character
		}

		return items[i].Range.Start.Line < items[j].Range.Start.Line
	})
}
