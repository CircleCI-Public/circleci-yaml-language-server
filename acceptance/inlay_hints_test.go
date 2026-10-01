package acceptance

import (
	"fmt"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

// hintsConfig declares an orb in each of the ways of naming its version, and
// a matrix.
const hintsConfig = `version: 2.1

orbs:
  major: circleci/go@1
  minor: circleci/go@1.7
  whole: circleci/go@1.7.1
  newest: circleci/go@volatile

jobs:
  build:
    parameters:
      os:
        type: string
    machine:
      image: ubuntu-2404:current
    steps:
      - run: echo << parameters.os >>

workflows:
  main:
    jobs:
      - build:
          matrix:
            parameters:
              os: [linux, macos]
`

func TestInlayHints(t *testing.T) {
	session := start(t, linkedProjectFake(t), hintsConfig, testToken)
	session.open(t, hintsConfig)

	// hints are the hints in a range, each as where it is and what it says.
	hints := func(t *testing.T, rng protocol.Range) []string {
		t.Helper()
		var found []protocol.InlayHint
		err := session.client.Call(protocol.MethodTextDocumentInlayHint, protocol.InlayHintParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: session.workspace.URI()},
			Range:        rng,
		}, &found)
		assert.NilError(t, err)

		got := []string{}
		for _, hint := range found {
			label, _ := hint.Label.(protocol.String)
			got = append(got, fmt.Sprintf("%d:%d %s", hint.Position.Line+1, hint.Position.Character, label))
		}
		return got
	}

	t.Run("an orb's version where it isn't a whole one, and a matrix's jobs", func(t *testing.T) {
		got := hints(t, protocol.Range{End: protocol.Position{Line: 99}})
		assert.Check(t, cmp.DeepEqual(got, []string{
			"4:22 → 1.12.0",
			"5:24 → 1.7.3",
			"7:30 → 4.0.0",
			"23:17 2 jobs",
		}))
	})

	t.Run("only in the range asked for", func(t *testing.T) {
		got := hints(t, protocol.Range{Start: protocol.Position{Line: 4}, End: protocol.Position{Line: 5}})
		assert.Check(t, cmp.DeepEqual(got, []string{"5:24 → 1.7.3"}))
	})
	t.Run("asks for hints again once an edit has fetched an orb", func(t *testing.T) {
		edited := strings.Replace(hintsConfig, "circleci/go@1.7\n", "circleci/go@0\n", 1)
		refreshes := session.client.InlayHintRefreshes()
		err := session.client.DidChange(session.workspace.URI(), 2, edited)
		assert.NilError(t, err)

		eventually(t, "a refresh", func() bool { return session.client.InlayHintRefreshes() > refreshes })
		got := hints(t, protocol.Range{Start: protocol.Position{Line: 4}, End: protocol.Position{Line: 5}})
		assert.Check(t, cmp.DeepEqual(got, []string{"5:22 → 0.1.0"}))
	})
}
