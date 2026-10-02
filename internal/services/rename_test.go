package languageservice

import (
	"errors"
	"strings"
	"testing"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestRename(t *testing.T) {
	docURI := uri.File("/repo/.circleci/config.yml")

	// renameIn renames what is at the first mention of marker in config, and
	// gives the names the rename replaces, by line.
	renameIn := func(t *testing.T, config, marker, newName string) (map[uint32][]string, error) {
		t.Helper()
		c := cache.New()
		c.FileCache.SetFile(cache.File{TextDocument: protocol.TextDocumentItem{URI: docURI, Text: config}})
		index := strings.Index(config, marker)
		assert.Assert(t, index >= 0, "no %q in the config", marker)

		edit, err := Rename(t.Context(), protocol.RenameParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
				Position:     position.FromIndex(index, []byte(config)),
			},
			NewName: newName,
		}, c, testHelpers.DefaultSettings())
		if err != nil {
			return nil, err
		}

		replaced := map[uint32][]string{}
		content := []byte(config)
		for _, change := range edit.Changes[docURI] {
			start := position.ToIndex(change.Range.Start, content)
			end := position.ToIndex(change.Range.End, content)
			replaced[change.Range.Start.Line] = append(replaced[change.Range.Start.Line], config[start:end])
		}
		return replaced, nil
	}

	const config = `version: 2.1

executors:
  linux:
    machine:
      image: ubuntu-2404:current

jobs:
  build:
    executor: linux
    steps:
      - checkout
  deploy:
    executor: linux
    steps:
      - checkout

workflows:
  main:
    jobs:
      - build:
          name: compile
      - deploy:
          requires: [compile]
`

	t.Run("leaves a requires naming a job by a name of its own", func(t *testing.T) {
		replaced, err := renameIn(t, config, "build:\n    executor", "make")
		assert.NilError(t, err)
		assert.Check(t, cmp.DeepEqual(replaced, map[uint32][]string{8: {"build"}, 20: {"build"}}))
	})

	t.Run("refuses a name that isn't valid", func(t *testing.T) {
		for _, name := range []string{"", "two words", "orb/job", "1st", `"quoted"`} {
			_, err := renameIn(t, config, "build:\n    executor", name)
			checkRenameError(t, err, jsonrpc2.InvalidParams, "is not a valid name")
		}
	})

	t.Run("refuses a name that is taken", func(t *testing.T) {
		_, err := renameIn(t, config, "build:\n    executor", "deploy")
		checkRenameError(t, err, codeRequestFailed, `already a job named "deploy"`)
	})

	t.Run("refuses where there is nothing to rename", func(t *testing.T) {
		_, err := renameIn(t, config, "checkout", "fetch")
		checkRenameError(t, err, codeRequestFailed, "no job, command or executor here")
	})

	t.Run("refuses rather than leave a mention out", func(t *testing.T) {
		// A parameter with the executor's name for its value leaves no way to
		// tell which is the executor's.
		ambiguous := strings.Replace(config, "  deploy:\n    executor: linux", `  deploy:
    executor:
      name: linux
      os: linux`, 1)
		ambiguous = strings.Replace(ambiguous, "executors:\n  linux:\n", `executors:
  linux:
    parameters:
      os:
        type: string
`, 1)

		_, err := renameIn(t, ambiguous, "linux:", "ubuntu")
		checkRenameError(t, err, codeRequestFailed, `can't rename executor "linux"`)
	})

	t.Run("prepares nothing where there is nothing to rename", func(t *testing.T) {
		c := cache.New()
		c.FileCache.SetFile(cache.File{TextDocument: protocol.TextDocumentItem{URI: docURI, Text: config}})

		placeholder, err := PrepareRename(t.Context(), protocol.PrepareRenameParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
				Position:     protocol.Position{Line: 0, Character: 2},
			},
		}, c, testHelpers.DefaultSettings())
		assert.Check(t, err)
		assert.Check(t, cmp.Nil(placeholder))
	})
}

// checkRenameError checks that err is the JSON-RPC error a rename that can't
// be done fails with.
func checkRenameError(t *testing.T, err error, code jsonrpc2.Code, message string) {
	t.Helper()
	var rpcErr *jsonrpc2.Error
	isRPC := errors.As(err, &rpcErr)
	if !assert.Check(t, isRPC, "error %v is not a JSON-RPC error", err) {
		return
	}
	assert.Check(t, cmp.Equal(rpcErr.Code, code))
	assert.Check(t, cmp.Contains(rpcErr.Message, message))
}
