package hover

import (
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestStepOutput(t *testing.T) {
	const config = `version: 2.1

functions:
  build: github.com/circleci-functions/build@v0.1.0

jobs:
  deploy:
    docker:
      - image: cimg/base:stable
    steps:
      - build:
          id: image
      - build/push:
          with:
            tag: ${{ steps.image.outputs.digest }}
            tags: ${{ steps.image.outputs.tags[0] }}
            other: ${{ steps.image.outputs.missing }}
            before: ${{ steps.later.outputs.digest }}
      - run: echo ${{ steps.image.outputs.digest }}
      - run:
          command: echo
          environment:
            DIGEST: ${{ steps.image.outputs.digest }}
      - save_cache:
          key: v1-${{ steps.image.outputs.tags[0] }}
          paths: [a]
      - restore_cache:
          keys:
            - v1-${{ steps.image.outputs.tags[0] }}-keys
      - save_cache:
          keys:
            - v2-${{ steps.image.outputs.digest }}
          paths: [a]
      - build:
          id: later
`
	fake := fakes.NewCircleCI(t)
	fake.AddFunction("fn-build", "github.com/circleci-functions/build", "Build an image.",
		fakes.FunctionVersion{ID: "ver-build", Version: "v0.1.0", Descriptor: map[string]any{
			"name": "build",
			"outputs": []map[string]any{
				{"name": "digest", "type": "string", "description": "Digest of the pushed image.", "secret": true},
				{"name": "tags", "type": "list"},
			},
			"commands": map[string]any{"push": map[string]any{}},
		}},
	)
	settings := testHelpers.SettingsForHost(fake.URL())

	doc, err := yamlparser.ParseFromContent([]byte(config), settings, uri.File("/config.yml"), protocol.Position{})
	assert.NilError(t, err)
	t.Cleanup(doc.Close)
	c := cache.New()

	lines := strings.Split(config, "\n")
	at := func(t *testing.T, text string) protocol.Position {
		t.Helper()
		for i, line := range lines {
			if column := strings.Index(line, text); column != -1 {
				return protocol.Position{Line: uint32(i), Character: uint32(column + len(text)/2)}
			}
		}
		t.Fatalf("no line holds %q", text)
		return protocol.Position{}
	}

	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "a secret output shows its type, that it is secret, and its description",
			text: "tag: ${{ steps.image.outputs.digest }}",
			want: "**digest** output of **build**\n\n(string, secret): Digest of the pushed image.",
		},
		{
			name: "a list item shows the list",
			text: "steps.image.outputs.tags[0]",
			want: "**tags** output of **build**\n\n(list)",
		},
		{
			name: "a run command's reference shows the output",
			text: "echo ${{ steps.image.outputs.digest }}",
			want: "**digest** output of **build**\n\n(string, secret): Digest of the pushed image.",
		},
		{
			name: "a save_cache key shows the output",
			text: "key: v1-${{ steps.image.outputs.tags[0] }}",
			want: "**tags** output of **build**\n\n(list)",
		},
		{
			name: "a restore_cache key in keys shows the output",
			text: "v1-${{ steps.image.outputs.tags[0] }}-keys",
			want: "**tags** output of **build**\n\n(list)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := StepOutput(t.Context(), doc, c, at(t, tt.text))
			assert.Assert(t, ok)
			assert.Check(t, cmp.Equal(got, tt.want))
		})
	}

	t.Run("nothing is shown", func(t *testing.T) {
		tests := []struct {
			name string
			text string
		}{
			{"for an output the function doesn't declare", "steps.image.outputs.missing"},
			{"for a step that comes later", "steps.later.outputs.digest"},
			{"outside where outputs are read", "DIGEST: ${{ steps.image.outputs.digest }}"},
			{"in a save_cache's keys", "v2-${{ steps.image.outputs.digest }}"},
			{"outside a reference", "- run: echo ${{"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, ok := StepOutput(t.Context(), doc, c, at(t, tt.text))
				assert.Check(t, !ok)
			})
		}
	})
}
