package acceptance

import (
	"fmt"
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	pos "github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

// highlightConfig mentions a job, a job's parameter, a pipeline parameter and
// an orb more than once each.
const highlightConfig = `version: 2.1

orbs:
  node: circleci/node@5.0.0

parameters:
  deploy:
    type: boolean
    default: false

jobs:
  build:
    parameters:
      os:
        type: string
    machine:
      image: ubuntu-2404:current
    steps:
      - node/install
      - run: echo << parameters.os >> << pipeline.parameters.deploy >>

workflows:
  main:
    when: << pipeline.parameters.deploy >>
    jobs:
      - build:
          os: linux
      - build:
          name: again
          os: macos
          requires: [build]
`

func TestDocumentHighlight(t *testing.T) {
	session := start(t, linkedProjectFake(t), highlightConfig, testToken)
	session.open(t, highlightConfig)
	doc := session.workspace.URI()

	// highlighted is each highlight for what is at the first mention of
	// marker, as its line and text.
	highlighted := func(t *testing.T, marker string) []string {
		t.Helper()
		var found []protocol.DocumentHighlight
		err := session.client.Call(protocol.MethodTextDocumentDocumentHighlight, protocol.DocumentHighlightParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: doc},
				Position:     at(t, highlightConfig, marker, 1),
			},
		}, &found)
		assert.NilError(t, err)

		content := []byte(highlightConfig)
		texts := []string{}
		for _, highlight := range found {
			start := pos.ToIndex(highlight.Range.Start, content)
			end := pos.ToIndex(highlight.Range.End, content)
			texts = append(texts, fmt.Sprintf("%d:%s", highlight.Range.Start.Line+1, highlightConfig[start:end]))
		}
		return texts
	}

	t.Run("a job, where it is defined and run", func(t *testing.T) {
		want := []string{"12:build", "26:build", "28:build", "31:build"}
		got := highlighted(t, "build:\n    parameters")
		assert.Check(t, cmp.DeepEqual(got, want))
		got = highlighted(t, "build]")
		assert.Check(t, cmp.DeepEqual(got, want))
	})

	t.Run("a job's parameter, where it is declared and used", func(t *testing.T) {
		want := []string{"14:os", "20:os"}
		got := highlighted(t, "os:\n        type")
		assert.Check(t, cmp.DeepEqual(got, want))
		got = highlighted(t, "parameters.os")
		assert.Check(t, cmp.DeepEqual(got, want))
	})

	t.Run("a pipeline parameter, where it is declared and used", func(t *testing.T) {
		want := []string{"7:deploy", "20:deploy", "24:deploy"}
		got := highlighted(t, "deploy:")
		assert.Check(t, cmp.DeepEqual(got, want))
		got = highlighted(t, "pipeline.parameters.deploy")
		assert.Check(t, cmp.DeepEqual(got, want))
	})

	t.Run("an orb, where it is declared and what is taken from it", func(t *testing.T) {
		want := []string{"4:node", "19:node"}
		got := highlighted(t, "node:")
		assert.Check(t, cmp.DeepEqual(got, want))
		got = highlighted(t, "node/install")
		assert.Check(t, cmp.DeepEqual(got, want))
	})

	t.Run("nothing where nothing is named", func(t *testing.T) {
		got := highlighted(t, "version")
		assert.Check(t, cmp.Len(got, 0))
	})
}
