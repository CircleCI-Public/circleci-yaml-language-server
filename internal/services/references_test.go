package languageservice

import (
	"os"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestReferences(t *testing.T) {
	c := cache.New()

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
	// superorb/superfunc doesn't exist, which is all the fake has to say.
	context := testHelpers.SettingsForHost(fakes.NewCircleCI(t).URL())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, _ := os.ReadFile(tt.args.filePath)
			c.FileCache.SetFile(cache.File{
				TextDocument: protocol.TextDocumentItem{
					URI:  uri.File(tt.args.filePath),
					Text: string(content),
				},
				Project:      circleci.Project{},
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

			got, err := References(params, c, context)

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

func TestReferencesOfAnExecutorNamedByAParameter(t *testing.T) {
	content := `version: 2.1
executors:
  small:
    docker:
      - image: cimg/base:current
  big:
    docker:
      - image: cimg/base:current
jobs:
  direct:
    executor: small
    steps: [checkout]
  through-a-parameter:
    parameters:
      e:
        type: executor
        default: small
    executor: << parameters.e >>
    steps: [checkout]
  not-a-parameter-of-its-own:
    executor: << pipeline.parameters.e >>
    steps: [checkout]
workflows:
  main:
    jobs:
      - direct
      - through-a-parameter:
          name: argument
          e: small
      - through-a-parameter:
          name: quoted
          e: "small"
      - through-a-parameter:
          name: another
          e: big
      - through-a-parameter:
          matrix:
            parameters:
              e: [big, small]
`
	file := uri.File("executor-parameter.yml")
	c := cache.New()
	c.FileCache.SetFile(cache.File{
		TextDocument: protocol.TextDocumentItem{URI: file, Text: content},
		Project:      circleci.Project{},
		EnvVariables: []string{},
	})

	// 0-based, on the name `small` under executors.
	params := protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: file},
			Position:     protocol.Position{Line: 2, Character: 4},
		},
	}
	got, err := References(params, c, testHelpers.DefaultSettings())
	assert.NilError(t, err)

	// Each is found on its own line, 0-based: the job's executor, the
	// parameter's default, the two arguments and the matrix value.
	lines := []uint32{}
	for _, location := range got {
		lines = append(lines, location.Range.Start.Line)
	}
	slices.Sort(lines)
	assert.Check(t, cmp.DeepEqual(lines, []uint32{10, 16, 28, 31, 38}))

	source := strings.Split(content, "\n")
	for _, location := range got {
		rng := location.Range
		text := source[rng.Start.Line][rng.Start.Character:rng.End.Character]
		assert.Check(t, cmp.Contains(text, "small"), "line %d", rng.Start.Line)
	}
}
