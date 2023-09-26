package languageservice

import (
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/services/complete"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/testHelpers"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestComplete(t *testing.T) {
	cwd, _ := os.Getwd()
	schemaPath, _ := filepath.Abs(cwd + "/../../schema.json")
	os.Setenv("SCHEMA_LOCATION", schemaPath)
	cache := utils.CreateCache()

	context := testHelpers.GetDefaultLsContext()

	parsedOrb, err := parser.ParseFromURI(
		uri.File(path.Join("./testdata/orb.yaml")),
		context,
	)

	if err != nil {
		panic(err)
	}

	builtInEnvsComplete := []protocol.CompletionItem{}
	for _, env := range complete.BUILT_IN_ENV {
		builtInEnvsComplete = append(builtInEnvsComplete, protocol.CompletionItem{
			Label:    env,
			Detail:   "Built-in environment variable",
			SortText: "C",
		})
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
		want    []protocol.CompletionItem
		wantErr bool
	}{
		{
			name: "Completion for job param's type",
			args: args{
				filePath: "./testdata/autocomplete1.yml",
				position: protocol.Position{
					Line:      7,
					Character: 22,
				},
			},
			want: []protocol.CompletionItem{
				{
					Label: "string",
				},
				{
					Label: "boolean",
				},
				{
					Label: "integer",
				},
				{
					Label: "enum",
				},
				{
					Label: "executor",
				},
				{
					Label: "steps",
				},
				{
					Label: "env_var_name",
				},
			},
		},
		{
			name: "Completion for job syntax",
			args: args{
				filePath: "./testdata/autocomplete1.yml",
				position: protocol.Position{
					Line:      4,
					Character: 8,
				},
			},
			want: []protocol.CompletionItem{
				{
					Label:      "description",
					InsertText: "description: ",
					Kind:       protocol.CompletionItemKindProperty,
				},
				{
					Label:      "executor",
					InsertText: "executor: ",
					Kind:       protocol.CompletionItemKindProperty,
				},
				{
					Label:      "resource_class",
					InsertText: "resource_class: ",
					Kind:       protocol.CompletionItemKindProperty,
				},
				{
					Label:      "shell",
					InsertText: "shell: ",
					Kind:       protocol.CompletionItemKindProperty,
				},
			},
		},
		{
			name: "Completion for job steps",
			args: args{
				filePath: "./testdata/autocomplete1.yml",
				position: protocol.Position{
					Line:      19,
					Character: 14,
				},
			},
			want: []protocol.CompletionItem{
				// User defined commands
				{
					Label: "dummyCommand",
				},
				// Itself (it can be called from itself)
				{
					Label: "terraform-init-plan",
				},
				// User defined job
				{
					Label: "dummyJob",
				},
				// Built-in steps
				{
					Label: "run",
				},
				{
					Label: "checkout",
				},
				{
					Label: "setup_remote_docker",
				},
				{
					Label: "save_cache",
				},
				{
					Label: "restore_cache",
				},
				{
					Label: "store_artifacts",
				},
				{
					Label: "store_test_results",
				},
				{
					Label: "persist_to_workspace",
				},
				{
					Label: "attach_workspace",
				},
				{
					Label: "add_ssh_keys",
				},
				{
					Label: "unless",
				},
				{
					Label: "when",
				},
				{
					Label: "superOrb/supermethod",
				},
			},
		},
		{
			name: "Completion for executors type",
			args: args{
				filePath: "./testdata/autocomplete1.yml",
				position: protocol.Position{
					Line:      33,
					Character: 8,
				},
			},
			want: []protocol.CompletionItem{
				{
					Label:      "docker",
					InsertText: "docker: ",
				},
				{
					Label:      "macos",
					InsertText: "macos: ",
				},
				{
					Label:      "windows",
					InsertText: "windows: ",
				},
				{
					Label:      "machine",
					InsertText: "machine: ",
				},
			},
		},
		{
			name: "Completion for executors machine image",
			args: args{
				filePath: "./testdata/autocomplete1.yml",
				position: protocol.Position{
					Line:      37,
					Character: 19,
				},
			},
			want: createCompletionItemForUbuntuImages(),
		},
		{
			name: "Completion for resource class",
			args: args{
				filePath: "./testdata/autocomplete1.yml",
				position: protocol.Position{
					Line:      41,
					Character: 24,
				},
			},
			want: []protocol.CompletionItem{
				{Label: "medium"},
				{Label: "macos.x86.medium.gen2"},
				{Label: "large"},
				{Label: "macos.m1.medium.gen1"},
				{Label: "macos.m1.large.gen1"},
				{Label: "macos.x86.metal.gen1"},
			},
		},
		{
			name: "Completion for executors reference in jobs",
			args: args{
				filePath: "./testdata/autocomplete1.yml",
				position: protocol.Position{
					Line:      14,
					Character: 18,
				},
			},
			want: []protocol.CompletionItem{
				{
					Label: "machineExec",
				},
				{
					Label: "resourceClass",
				},
				{
					Label: "superOrb/default",
				},
			},
		},
		{
			name: "Completion for commands",
			args: args{
				filePath: "./testdata/autocomplete1.yml",
				position: protocol.Position{
					Line:      30,
					Character: 8,
				},
			},
			want: []protocol.CompletionItem{
				{
					Label:      "steps",
					InsertText: "steps: ",
				},
				{
					Label:      "description",
					InsertText: "description: ",
				},
			},
		},
		{
			name: "Completion for env variables",
			args: args{
				filePath: "./testdata/autocomplete1.yml",
				position: protocol.Position{
					Line:      28,
					Character: 37,
				},
			},
			want: builtInEnvsComplete,
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

			param := protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{
						URI: uri.File(tt.args.filePath),
					},
					Position: tt.args.position,
				},
			}

			got, err := Complete(param, cache, context)
			if (err != nil) != tt.wantErr {
				t.Errorf("Complete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// We don't care about the order of the items,
			// so we sort them before comparing to avoid the order
			// being the reason the test doesn't pass.
			sortCompleteItem(got.Items)
			sortCompleteItem(tt.want)

			if !reflect.DeepEqual(got.Items, tt.want) {
				t.Errorf("Complete(): %s = %v, want %v", tt.name, got.Items, tt.want)
			}
		})
	}
}

func sortCompleteItem(items []protocol.CompletionItem) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].Label < items[j].Label
	})
}

func createCompletionItemForUbuntuImages() []protocol.CompletionItem {
	completeItems := make([]protocol.CompletionItem, 0)
	for _, image := range utils.ValidARMOrMachineImagesUbuntu2004 {
		completeItems = append(completeItems, protocol.CompletionItem{
			Label: image,
		})
	}
	for _, image := range utils.ValidARMOrMachineImagesUbuntu2204 {
		completeItems = append(completeItems, protocol.CompletionItem{
			Label: image,
		})
	}
	return completeItems
}
