package languageservice

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/services/complete"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestComplete(t *testing.T) {
	c := cache.New()

	context := testHelpers.DefaultSettings()

	c.MachineOfferingsCache.Set(&circleci.Offerings{
		Linux: map[string][]string{
			"medium": {"ubuntu-2404:current", "ubuntu-2204:current"},
			"large":  {"ubuntu-2404:current", "ubuntu-2204:current"},
		},
		Windows: map[string][]string{"windows.medium": {"windows-server-2022-gui:current"}},
		MacOS: map[string][]string{
			"m4pro.medium": {"xcode:26.5.0"},
			"m4pro.large":  {"xcode:26.5.0"},
		},
	})

	orbPath, err := filepath.Abs("./testdata/orb.yaml")
	assert.NilError(t, err)

	parsedOrb, err := parser.ParseFromURI(
		uri.File(orbPath),
		context,
	)
	if err != nil {
		panic(err)
	}

	builtInEnvsComplete := []protocol.CompletionItem{}
	for _, env := range complete.BUILT_IN_ENV {
		builtInEnvsComplete = append(builtInEnvsComplete, protocol.CompletionItem{
			Label:    env,
			Detail:   protocol.NewOptional("Built-in environment variable"),
			SortText: protocol.NewOptional("C"),
		})
	}

	c.OrbCache.SetOrb(&ast.OrbInfo{
		OrbParsedAttributes: parsedOrb.ToOrbParsedAttributes(),
		RemoteInfo: ast.RemoteOrbInfo{
			FilePath: uri.File(orbPath).FsPath(),
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
				// The job has executor, parameters, working_directory and steps
				// already, and the keys of other job types are left out.
				{
					Label:      "description",
					InsertText: protocol.NewOptional("description: "),
					Kind:       protocol.CompletionItemKindProperty,
				},
				{
					Label:      "resource_class",
					InsertText: protocol.NewOptional("resource_class: "),
					Kind:       protocol.CompletionItemKindProperty,
				},
				{
					Label:      "shell",
					InsertText: protocol.NewOptional("shell: "),
					Kind:       protocol.CompletionItemKindProperty,
				},
				{
					Label:      "environment",
					InsertText: protocol.NewOptional("environment:\n\t"),
					Kind:       protocol.CompletionItemKindProperty,
				},
				{
					Label:      "parallelism",
					InsertText: protocol.NewOptional("parallelism: "),
					Kind:       protocol.CompletionItemKindProperty,
				},
				{
					Label:      "circleci_ip_ranges",
					InsertText: protocol.NewOptional("circleci_ip_ranges: "),
					Kind:       protocol.CompletionItemKindProperty,
				},
				{
					Label:      "retention",
					InsertText: protocol.NewOptional("retention:\n\t"),
					Kind:       protocol.CompletionItemKindProperty,
				},
				{
					Label:      "type",
					InsertText: protocol.NewOptional("type: "),
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
					Label: "install_signing_bundle",
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
					Label: "with_tool_cache",
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
					InsertText: protocol.NewOptional("docker: "),
				},
				{
					Label:      "macos",
					InsertText: protocol.NewOptional("macos: "),
				},
				{
					Label:      "machine",
					InsertText: protocol.NewOptional("machine: "),
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
			want: createCompletionItemForLabels(c.Offerings(context.Api).MachineImages()),
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
			want: createCompletionItemForLabels(c.Offerings(context.Api).MacOSResourceClasses()),
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
					InsertText: protocol.NewOptional("steps: "),
				},
				{
					Label:      "description",
					InsertText: protocol.NewOptional("description: "),
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
			c.FileCache.SetFile(cache.File{
				TextDocument: protocol.TextDocumentItem{
					URI:  uri.File(tt.args.filePath),
					Text: string(content),
				},
				Project:      circleci.Project{},
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

			got, err := Complete(param, c, context)
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

func createCompletionItemForLabels(labels []string) []protocol.CompletionItem {
	completeItems := make([]protocol.CompletionItem, len(labels))
	for i, label := range labels {
		completeItems[i].Label = label
	}
	return completeItems
}
