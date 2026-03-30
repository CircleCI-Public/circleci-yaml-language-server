package languageservice

import (
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/services/definition"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/testHelpers"
	utils "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestDefinition(t *testing.T) {
	cwd, _ := os.Getwd()
	schemaPath, _ := filepath.Abs(cwd + "/../../schema.json")
	os.Setenv("SCHEMA_LOCATION", schemaPath)
	cache := utils.CreateCache()

	context := testHelpers.GetDefaultLsContext()
	parsedOrb, err := parser.ParseFromURI(uri.File(path.Join("./testdata/orb.yaml")), context)

	if err != nil {
		panic(err)
	}

	cache.OrbCache.SetOrb(&ast.OrbInfo{
		OrbParsedAttributes: parsedOrb.ToOrbParsedAttributes(),
		RemoteInfo: ast.RemoteOrbInfo{
			FilePath: uri.File(path.Join("./testdata/orb.yaml")).Filename(),
		},
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
			name: "Definition for job invocation",
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
			cache.FileCache.SetFile(utils.CachedFile{
				TextDocument: protocol.TextDocumentItem{
					URI:  uri.File(tt.args.filePath),
					Text: string(content),
				},
				Project:      utils.Project{},
				EnvVariables: make([]string, 0),
			})

			params := protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: uri.File(tt.args.filePath),
					},
					Position: tt.args.position,
				},
			}

			got, err := Definition(params, cache, context)
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

func TestDefinitionForLocalOrbsCommand(t *testing.T) {
	fileURI := uri.File("some-uri")
	yaml := `version: 2.1

orbs:
  local:
    commands:
      cmd:
        steps:
          - run: echo << parameters.target >>
    jobs:
      job:
        docker:
          - image: cimg/node:21.6.1
        steps:
          - cmd`
	context := testHelpers.GetDefaultLsContext()

	doc, err := parser.ParseFromContent([]byte(yaml), context, fileURI, protocol.Position{})
	assert.Nil(t, err)

	def := definition.DefinitionStruct{Cache: utils.CreateCache(), Params: protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: fileURI,
			},
			// Position is the job step `cmd`
			Position: protocol.Position{
				Line:      13,
				Character: 14,
			},
		},
	}, Doc: doc}
	locations, err := def.Definition()
	assert.Nil(t, err)
	assert.Equal(t, locations, []protocol.Location{
		{
			URI: fileURI,
			Range: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 6},
				End:   protocol.Position{Line: 7, Character: 45},
			},
		},
	})
}

func TestDefinitionForLocalOrbsExecutor(t *testing.T) {
	fileURI := uri.File("some-uri")
	yaml := `version: 2.1

orbs:
  local:
    executors:
      executor:
        docker:
          - image: cimg/base:2024.01
    jobs:
      job:
        executor: executor
        steps:
          - run: echo "Hello World"`
	context := testHelpers.GetDefaultLsContext()

	doc, err := parser.ParseFromContent([]byte(yaml), context, fileURI, protocol.Position{})
	assert.Nil(t, err)

	def := definition.DefinitionStruct{Cache: utils.CreateCache(), Params: protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: fileURI,
			},
			// Position is the job step `cmd`
			Position: protocol.Position{
				Line:      10,
				Character: 24,
			},
		},
	}, Doc: doc}
	locations, err := def.Definition()
	assert.Nil(t, err)
	assert.Equal(t, locations, []protocol.Location{
		{
			URI: fileURI,
			Range: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 6},
				End:   protocol.Position{Line: 7, Character: 36},
			},
		},
	})
}

// Tests for goto-definition on job/job-group invocations in workflows and job-groups.
// The workflow and job-group tests come in matched pairs to show they behave identically.
//
// Shared fixture (line numbers for reference):
//
//  0: version: 2.1
//  1:
//  2: jobs:
//  3:   build:              <-- job definition
//  4:     docker:
//  5:       - image: cimg/base:stable
//  6:     steps:
//  7:       - run: echo "build"
//  8:   deploy:             <-- job definition
//  9:     docker:
// 10:       - image: cimg/base:stable
// 11:     steps:
// 12:       - run: echo "deploy"
// 13:
// 14: job-groups:
// 15:   my-group:           <-- job-group definition
// 16:     jobs:
// 17:       - build         <-- job invocation inside job-group
// 18:       - deploy:       <-- job invocation inside job-group
// 19:           requires:
// 20:             - build   <-- requires inside job-group
// 21:
// 22: workflows:
// 23:   main:
// 24:     jobs:
// 25:       - my-group      <-- job-group invocation inside workflow
// 26:       - build         <-- job invocation inside workflow
// 27:       - deploy:       <-- job invocation inside workflow
// 28:           requires:
// 29:             - build   <-- requires inside workflow

var jobGroupDefinitionFixture = `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "build"
  deploy:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "deploy"

job-groups:
  my-group:
    jobs:
      - build
      - deploy:
          requires:
            - build

workflows:
  main:
    jobs:
      - my-group
      - build
      - deploy:
          requires:
            - build`

func parseJobGroupDefinitionFixture(t *testing.T) (parser.YamlDocument, protocol.URI) {
	t.Helper()
	fileURI := uri.File("some-uri")
	context := testHelpers.GetDefaultLsContext()
	doc, err := parser.ParseFromContent([]byte(jobGroupDefinitionFixture), context, fileURI, protocol.Position{})
	assert.Nil(t, err)
	return doc, fileURI
}

func definitionAt(t *testing.T, doc parser.YamlDocument, fileURI protocol.URI, line, char uint32) []protocol.Location {
	t.Helper()
	def := definition.DefinitionStruct{Cache: utils.CreateCache(), Params: protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: fileURI},
			Position:     protocol.Position{Line: line, Character: char},
		},
	}, Doc: doc}
	locations, err := def.Definition()
	assert.Nil(t, err)
	return locations
}

// Pair 1: job name → jumps to job definition

func TestDefinition_WorkflowJobName_GoesToJobDef(t *testing.T) {
	doc, fileURI := parseJobGroupDefinitionFixture(t)
	// Goto def on "build" on line 26 in workflow
	locations := definitionAt(t, doc, fileURI, 26, 8)
	assert.NotEmpty(t, locations)
	assert.Equal(t, uint32(3), locations[0].Range.Start.Line, "should jump to jobs.build definition")
}

func TestDefinition_JobGroupJobName_GoesToJobDef(t *testing.T) {
	doc, fileURI := parseJobGroupDefinitionFixture(t)
	// Goto def on "build" on line 17 in job-group
	locations := definitionAt(t, doc, fileURI, 17, 8)
	assert.NotEmpty(t, locations)
	assert.Equal(t, uint32(3), locations[0].Range.Start.Line, "should jump to jobs.build definition")
}

// Pair 2: requires entry → jumps to job-invocation in same block

func TestDefinition_WorkflowRequires_GoesToJobInvocation(t *testing.T) {
	doc, fileURI := parseJobGroupDefinitionFixture(t)
	// Goto def on "build" in requires on line 29 in workflow
	locations := definitionAt(t, doc, fileURI, 29, 14)
	assert.NotEmpty(t, locations)
	assert.Equal(t, uint32(26), locations[0].Range.Start.Line, "should jump to the 'build' job-invocation in the workflow")
}

func TestDefinition_JobGroupRequires_GoesToJobInvocation(t *testing.T) {
	doc, fileURI := parseJobGroupDefinitionFixture(t)
	// Goto def on "build" in requires on line 20 in job-group
	locations := definitionAt(t, doc, fileURI, 20, 14)
	assert.NotEmpty(t, locations)
	assert.Equal(t, uint32(17), locations[0].Range.Start.Line, "should jump to the 'build' job-invocation in the job-group")
}

// Unique to job-groups: job-group name in a workflow → jumps to job-group definition

func TestDefinition_WorkflowJobGroupName_GoesToJobGroupDef(t *testing.T) {
	doc, fileURI := parseJobGroupDefinitionFixture(t)
	// Goto def on "my-group" on line 25 in workflow
	locations := definitionAt(t, doc, fileURI, 25, 8)
	assert.NotEmpty(t, locations)
	assert.Equal(t, uint32(15), locations[0].Range.Start.Line, "should jump to job-groups.my-group definition")
}

// Tests for goto-definition when a job invocation uses name: to rename itself.
// The requires entry uses the renamed name, and goto-def should still resolve.
//
// Fixture (line numbers for reference):
//
//  0: version: 2.1
//  1:
//  2: jobs:
//  3:   build:
//  4:     docker:
//  5:       - image: cimg/base:stable
//  6:     steps:
//  7:       - run: echo "build"
//  8:   deploy:
//  9:     docker:
// 10:       - image: cimg/base:stable
// 11:     steps:
// 12:       - run: echo "deploy"
// 13:
// 14: workflows:
// 15:   main:
// 16:     jobs:
// 17:       - build:
// 18:           name: build-renamed
// 19:       - deploy:
// 20:           requires:
// 21:             - build-renamed

var renamedJobDefinitionFixture = `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "build"
  deploy:
    docker:
      - image: cimg/base:stable
    steps:
      - run: echo "deploy"

workflows:
  main:
    jobs:
      - build:
          name: build-renamed
      - deploy:
          requires:
            - build-renamed`

func TestDefinition_WorkflowRequiresRenamedJob_GoesToJobInvocation(t *testing.T) {
	fileURI := uri.File("some-uri")
	context := testHelpers.GetDefaultLsContext()
	doc, err := parser.ParseFromContent([]byte(renamedJobDefinitionFixture), context, fileURI, protocol.Position{})
	assert.Nil(t, err)

	// Goto def on "build-renamed" in requires on line 21
	locations := definitionAt(t, doc, fileURI, 21, 14)
	assert.NotEmpty(t, locations, "should resolve goto-def for renamed job in requires")
	assert.Equal(t, uint32(17), locations[0].Range.Start.Line, "should jump to the 'build' job-invocation (which has name: build-renamed)")
}
