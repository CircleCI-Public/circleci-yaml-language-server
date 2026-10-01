package acceptance

import (
	"slices"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	pos "github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

// renameConfig uses a job, a command and an executor in each of the ways a
// rename has to follow.
const renameConfig = `version: 2.1

executors:
  linux:
    machine:
      image: ubuntu-2404:current

commands:
  greet:
    steps:
      - run: echo hello

jobs:
  build:
    executor: linux
    parameters:
      setup:
        type: steps
        default: [greet]
    steps:
      - steps: << parameters.setup >>
      - greet
      - when:
          condition: true
          steps:
            - greet
  lint:
    executor:
      name: linux
    parameters:
      os:
        type: string
    steps:
      - run: echo << parameters.os >>

workflows:
  main:
    jobs:
      - build:
          pre-steps:
            - greet
          setup:
            - greet
      - lint:
          matrix:
            parameters:
              os: [a, b]
          requires: [build]
      - build:
          name: deploy
          requires:
            - lint-a
            - "lint"
`

// at is where the nth mention of text in config starts, counting from 1.
func at(t *testing.T, config, text string, nth int) protocol.Position {
	t.Helper()
	index := -1
	for range nth {
		next := strings.Index(config[index+1:], text)
		assert.Assert(t, next >= 0, "no mention %d of %q", nth, text)
		index += 1 + next
	}
	return pos.FromIndex(index, []byte(config))
}

// applyEdits is config once edits are made to it.
func applyEdits(config string, edits []protocol.TextEdit) string {
	edits = slices.Clone(edits)
	slices.SortFunc(edits, func(a, b protocol.TextEdit) int { return pos.Compare(b.Range.Start, a.Range.Start) })
	content := []byte(config)
	for _, edit := range edits {
		start := pos.ToIndex(edit.Range.Start, content)
		end := pos.ToIndex(edit.Range.End, content)
		content = slices.Concat(content[:start], []byte(edit.NewText), content[end:])
	}
	return string(content)
}

func TestRename(t *testing.T) {
	session := start(t, linkedProjectFake(t), renameConfig, testToken)
	diagnostics := session.open(t, renameConfig)
	assert.Assert(t, cmp.Len(diagnostics, 0))
	doc := session.workspace.URI()

	rename := func(t *testing.T, at protocol.Position, newName string) string {
		t.Helper()
		var edit protocol.WorkspaceEdit
		err := session.client.Call(protocol.MethodTextDocumentRename, protocol.RenameParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: doc},
				Position:     at,
			},
			NewName: newName,
		}, &edit)
		assert.NilError(t, err)
		return applyEdits(renameConfig, edit.Changes[doc])
	}

	t.Run("a job, from where a workflow runs it", func(t *testing.T) {
		got := rename(t, at(t, renameConfig, "lint:\n          matrix", 1), "check")

		want := strings.NewReplacer(
			"  lint:\n    executor", "  check:\n    executor",
			"      - lint:", "      - check:",
			"- lint-a", "- check-a",
			`- "lint"`, `- "check"`,
		).Replace(renameConfig)
		assert.Check(t, cmp.Equal(got, want))

		t.Run("which is still a valid config", func(t *testing.T) {
			diagnostics := session.open(t, got)
			assert.Check(t, cmp.Len(diagnostics, 0))
		})
	})

	t.Run("a command, from where it is defined", func(t *testing.T) {
		got := rename(t, at(t, renameConfig, "greet:", 1), "say-hello")

		want := strings.ReplaceAll(renameConfig, "greet", "say-hello")
		assert.Check(t, cmp.Equal(got, want))

		t.Run("which is still a valid config", func(t *testing.T) {
			diagnostics := session.open(t, got)
			assert.Check(t, cmp.Len(diagnostics, 0))
		})
	})

	t.Run("an executor, from where a job uses it", func(t *testing.T) {
		got := rename(t, at(t, renameConfig, "linux", 3), "ubuntu")

		want := strings.ReplaceAll(renameConfig, "linux", "ubuntu")
		assert.Check(t, cmp.Equal(got, want))

		t.Run("which is still a valid config", func(t *testing.T) {
			diagnostics := session.open(t, got)
			assert.Check(t, cmp.Len(diagnostics, 0))
		})
	})

	t.Run("prepares by giving the name under the cursor", func(t *testing.T) {
		session.open(t, renameConfig)
		var placeholder protocol.PrepareRenamePlaceholder
		err := session.client.Call(protocol.MethodTextDocumentPrepareRename, protocol.PrepareRenameParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: doc},
				Position:     at(t, renameConfig, `lint"`, 1),
			},
		}, &placeholder)
		assert.NilError(t, err)

		start := at(t, renameConfig, `lint"`, 1)
		end := start
		end.Character += uint32(len("lint"))
		assert.Check(t, cmp.DeepEqual(placeholder, protocol.PrepareRenamePlaceholder{
			Range:       protocol.Range{Start: start, End: end},
			Placeholder: "lint",
		}))
	})
}
