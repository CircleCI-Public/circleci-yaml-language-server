package complete

import (
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestCompleteStepOutputs(t *testing.T) {
	fake := fakes.NewCircleCI(t)
	fake.AddFunction("fn-build", "github.com/circleci-functions/build", "Build an image.",
		fakes.FunctionVersion{ID: "ver-build", Version: "v0.1.0", Descriptor: map[string]any{
			"name": "build",
			"outputs": []map[string]any{
				{"name": "digest", "type": "string", "description": "Digest of the pushed image.", "secret": true},
				{"name": "tags", "type": "list"},
				{"name": "no-type"},
				{"type": "string"},
			},
		}},
	)
	fake.AddFunction("fn-plain", "github.com/circleci-functions/plain", "Does something.",
		fakes.FunctionVersion{ID: "ver-plain", Version: "v0.1.0", Descriptor: map[string]any{"name": "plain"}},
	)
	settings := testHelpers.SettingsForHost(fake.URL())

	const marker = "§"
	complete := func(t *testing.T, steps string) []protocol.CompletionItem {
		t.Helper()
		config := `version: 2.1

functions:
  build: github.com/circleci-functions/build@v0.1.0
  plain: github.com/circleci-functions/plain@v0.1.0

jobs:
  deploy:
    docker:
      - image: cimg/base:stable
    steps:
      - build:
          id: image
      - plain:
          id: other
      - build
` + steps + `
workflows:
  main:
    jobs:
      - deploy
`
		before, after, found := strings.Cut(config, marker)
		assert.Assert(t, found, "no cursor")
		lines := strings.Split(before, "\n")
		pos := protocol.Position{Line: uint32(len(lines) - 1), Character: uint32(len(lines[len(lines)-1]))}
		return completionItemsWith(t, settings, cache.New(), before+after, pos)
	}
	labels := func(items []protocol.CompletionItem) []string {
		got := []string{}
		for _, item := range items {
			got = append(got, item.Label)
		}
		return got
	}

	places := []struct {
		name  string
		steps string
	}{
		{"a function step's with value", "      - build:\n          with:\n            tag: ${{ steps.§\n"},
		{"a run step's command", "      - run: echo ${{ steps.§\n"},
		{"a run step's multi-line command", "      - run:\n          command: |\n            echo hi\n            echo ${{ steps.§\n"},
		{"a save_cache key", "      - save_cache:\n          key: v1-${{ steps.§\n          paths: [a]\n"},
		{"a restore_cache key", "      - restore_cache:\n          key: v1-${{ steps.§\n"},
		{"a restore_cache key in a list", "      - restore_cache:\n          keys:\n            - v1-${{ steps.§\n            - v2\n"},
	}
	for _, place := range places {
		t.Run("the ids of earlier function steps are offered in "+place.name, func(t *testing.T) {
			items := complete(t, place.steps)
			assert.Check(t, cmp.DeepEqual(labels(items), []string{"image", "other"}))
			image := findItem(t, items, "image")
			assert.Check(t, cmp.Equal(image.Detail, protocol.NewOptional("build")))
			assert.Check(t, cmp.Equal(image.InsertText, protocol.NewOptional("image.outputs.")))
		})
	}

	t.Run("an id being written is offered the ids", func(t *testing.T) {
		items := complete(t, "      - run: echo ${{ steps.im§\n")
		assert.Check(t, cmp.DeepEqual(labels(items), []string{"image", "other"}))
	})

	t.Run("a function step is offered the ids of the steps before it, not its own", func(t *testing.T) {
		items := complete(t, "      - build:\n          id: later\n          with:\n            tag: ${{ steps.§\n      - plain:\n          id: after\n")
		assert.Check(t, cmp.DeepEqual(labels(items), []string{"image", "other"}))
	})

	t.Run("a function's output names are offered with their type and description", func(t *testing.T) {
		items := complete(t, "      - run: echo ${{ steps.image.outputs.§\n")
		assert.Check(t, cmp.DeepEqual(labels(items), []string{"digest", "tags"}))

		digest := findItem(t, items, "digest")
		assert.Check(t, cmp.Equal(digest.Kind, protocol.CompletionItemKindVariable))
		assert.Check(t, cmp.Equal(digest.Detail, protocol.NewOptional("string, secret")))
		assert.Check(t, cmp.Equal(digest.InsertText, protocol.NewOptional("digest }}")))
		assert.Check(t, cmp.DeepEqual(digest.Documentation,
			&protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: "Digest of the pushed image."}))

		tags := findItem(t, items, "tags")
		assert.Check(t, cmp.Equal(tags.Detail, protocol.NewOptional("list")))
		assert.Check(t, cmp.Nil(tags.Documentation))
	})

	t.Run("an output name being written is offered the names", func(t *testing.T) {
		items := complete(t, "      - run: echo ${{ steps.image.outputs.di§\n")
		assert.Check(t, cmp.DeepEqual(labels(items), []string{"digest", "tags"}))
	})

	t.Run("a reference already closed isn't closed again", func(t *testing.T) {
		items := complete(t, "      - run: echo ${{ steps.image.outputs.§ }}\n")
		assert.Check(t, cmp.Equal(findItem(t, items, "digest").InsertText, protocol.NewOptional("digest")))
	})

	t.Run("the reference is closed unless what follows the cursor closes it", func(t *testing.T) {
		tests := []struct {
			name   string
			rest   string
			insert string
		}{
			{"nothing follows", "", "digest }}"},
			{"it is closed", " }}", "digest"},
			{"it is closed with no space", "}}", "digest"},
			{"the rest of the name is followed by the close", "est }}", "digest"},
			{"a template follows", " - {{ checksum \"go.sum\" }}", "digest }}"},
			{"another reference follows", " ${{ steps.image.outputs.tags[0] }}", "digest }}"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				items := complete(t, "      - run: echo ${{ steps.image.outputs.§"+tt.rest+"\n")
				assert.Check(t, cmp.Equal(findItem(t, items, "digest").InsertText, protocol.NewOptional(tt.insert)))
			})
		}
	})

	t.Run("nothing is offered", func(t *testing.T) {
		tests := []struct {
			name  string
			steps string
		}{
			{"for a function that declares no outputs", "      - run: echo ${{ steps.other.outputs.§\n"},
			{"for an id that isn't a step before", "      - run: echo ${{ steps.nope.outputs.§\n"},
			{"for the id of a step after", "      - run: echo ${{ steps.after.outputs.§\n      - plain:\n          id: after\n"},
			{"in a run step's environment", "      - run:\n          command: echo\n          environment:\n            DIGEST: ${{ steps.§\n"},
			{"in a function step's id", "      - build:\n          id: ${{ steps.§\n"},
			{"in a save_cache keys", "      - save_cache:\n          keys:\n            - v1-${{ steps.§\n          paths: [a]\n"},
			{"in a save_cache keys on one line", "      - save_cache:\n          keys: v1-${{ steps.§\n          paths: [a]\n"},
			{"in a restore_cache path", "      - restore_cache:\n          paths:\n            - ${{ steps.§\n"},
			{"in a save_cache path", "      - save_cache:\n          key: v1\n          paths:\n            - ${{ steps.§\n"},
			{"in a reference to another kind of value", "      - run: echo ${{ github.§\n"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Check(t, cmp.DeepEqual(labels(complete(t, tt.steps)), []string{}))
			})
		}
	})
}
