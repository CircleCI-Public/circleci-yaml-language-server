package languageservice

import (
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/tsalloc"
)

// leakConfig reaches most of what a request parses: a local orb, which is
// parsed as a document of its own, parameters, an anchor and a machine
// executor. It stays away from registry orbs and Docker images, which would
// reach real services.
const leakConfig = `version: 2.1

orbs:
  local:
    commands:
      greet:
        steps:
          - run: echo hello

parameters:
  greeting:
    type: string
    default: hello

executors:
  linux: &linux
    machine:
      image: ubuntu-2404:current

jobs:
  build:
    executor: linux
    steps:
      - checkout
      - local/greet
      - run: echo << pipeline.parameters.greeting >>

workflows:
  main:
    jobs:
      - build
`

// Every request parses the document it is about, and the official tree-sitter
// bindings free a tree only when it is closed. So each request must close what
// it parsed, or the server leaks a tree per keystroke.
func TestRequestsCloseTheTreesTheyParse(t *testing.T) {
	fake := fakes.NewCircleCI(t)
	settings := testHelpers.SettingsForHost(fake.URL())

	docURI := uri.File("/workspace/.circleci/config.yml")
	c := cache.New()
	c.FileCache.SetFile(cache.File{
		TextDocument: protocol.TextDocumentItem{URI: docURI, Text: leakConfig},
	})

	document := protocol.TextDocumentIdentifier{URI: docURI}
	// On the step list of the build job, which is where completion edits the
	// document into variants and parses each of them.
	inSteps := protocol.Position{Line: 23, Character: 6}
	onGreet := protocol.Position{Line: 24, Character: 10}

	requests := []struct {
		name string
		run  func()
	}{
		{"diagnostics of a file", func() {
			_, _ = DiagnosticFile(docURI, c, settings, "")
		}},
		{"diagnostics of a string", func() {
			_, _ = DiagnosticString(leakConfig, c, settings, "")
		}},
		{"completion", func() {
			_, _ = Complete(protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{TextDocument: document, Position: inSteps},
			}, c, settings)
		}},
		{"hover", func() {
			_, _ = Hover(protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{TextDocument: document, Position: onGreet},
			}, c, settings)
		}},
		{"definition", func() {
			_, _ = Definition(protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{TextDocument: document, Position: onGreet},
			}, c, settings)
		}},
		{"references", func() {
			_, _ = References(protocol.ReferenceParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{TextDocument: document, Position: onGreet},
			}, c, settings)
		}},
		{"document symbols", func() {
			_, _ = DocumentSymbols(protocol.DocumentSymbolParams{TextDocument: document}, c, settings)
		}},
		{"semantic tokens", func() {
			_ = SemanticTokens(protocol.SemanticTokensParams{TextDocument: document}, c, settings)
		}},
	}

	for _, request := range requests {
		t.Run(request.name, func(t *testing.T) {
			leaked := tsalloc.Track(t)

			request.run()

			assert.Check(t, cmp.Equal(leaked(), int64(0)), "tree-sitter allocations left open")
		})
	}
}
