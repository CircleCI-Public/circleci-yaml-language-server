package languageservice

import (
	"fmt"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

const semanticAliasesConfig = `version: 2.1

orbs:
  outer:
    commands:
      greet:
        steps:
          - run: echo hello
  inner:
    orbs:
      tools:
        commands:
          install:
            steps:
              - run: echo install
    commands:
      install: tools/install
      quoted-install: "tools/install"
      outer-greet: outer/greet
      local: install
    jobs:
      build:
        machine:
          image: ubuntu-2404:current
        steps:
          - install
          - tools/install

commands:
  greet: 'outer/greet'
  undeclared: nope/greet

jobs:
  test:
    machine:
      image: ubuntu-2404:current
    steps:
      - greet

workflows:
  main:
    jobs:
      - test
      - inner/build
`

// tokenTypes are the names of the legend's token types, in its order.
var tokenTypes = []string{"keyword", "namespace", "class", "comment", "function"}

// semanticTokens decodes the tokens of content, as "line:column type text".
func semanticTokens(t *testing.T, content string) []string {
	t.Helper()

	c := cache.New()
	file := uri.File("/workspace/.circleci/config.yml")
	c.FileCache.SetFile(cache.File{TextDocument: protocol.TextDocumentItem{URI: file, Text: content}})
	params := protocol.SemanticTokensParams{TextDocument: protocol.TextDocumentIdentifier{URI: file}}
	data := SemanticTokens(t.Context(), params, c, testHelpers.DefaultSettings()).Data
	assert.Assert(t, cmp.Equal(len(data)%5, 0), "tokens come in fives")

	lines := strings.Split(content, "\n")
	tokens := []string{}
	var line, column uint32
	for i := 0; i < len(data); i += 5 {
		if data[i] != 0 {
			column = 0
		}
		line += data[i]
		column += data[i+1]
		text := lines[line][column : column+data[i+2]]
		tokens = append(tokens, fmt.Sprintf("%d:%d %s %s", line, column, tokenTypes[data[i+3]], text))
	}
	return tokens
}

func TestSemanticTokensOfAliases(t *testing.T) {
	tokens := semanticTokens(t, semanticAliasesConfig)

	on := func(line uint32) []string {
		res := []string{}
		for _, token := range tokens {
			if strings.HasPrefix(token, fmt.Sprintf("%d:", line)) {
				res = append(res, token)
			}
		}
		return res
	}

	t.Run("an inline orb's alias of its own orb's command", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(on(16), []string{"16:15 namespace tools/", "16:21 keyword install"}))
	})

	t.Run("a quoted alias", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(on(17), []string{"17:23 namespace tools/", "17:29 keyword install"}))
		assert.Check(t, cmp.DeepEqual(on(29), []string{"29:10 namespace outer/", "29:16 keyword greet"}))
	})

	t.Run("an inline orb's alias of the config's orb", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(on(18), []string{}))
	})

	t.Run("an alias of a local command", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(on(19), []string{}))
	})

	t.Run("an alias of an orb nothing declares", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(on(30), []string{}))
	})

	t.Run("an inline orb's own orbs", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(on(10), []string{"10:6 namespace tools"}))
		assert.Check(t, cmp.DeepEqual(on(26), []string{"26:12 namespace tools/", "26:18 keyword install"}))
	})

	t.Run("no token twice", func(t *testing.T) {
		seen := map[string]int{}
		for _, token := range tokens {
			seen[token]++
		}
		for token, count := range seen {
			assert.Check(t, cmp.Equal(count, 1), "token %q", token)
		}
	})
}

func TestSemanticTokensOfSections(t *testing.T) {
	tokens := semanticTokens(t, `version: 2.1

jobs:
  build:
    parameters:
      loud:
        type: boolean
        default: false
    docker:
      - image: cimg/base:current
    working_directory: <<# parameters.loud >>/tmp<</ parameters.loud >><<^ parameters.loud >>/src<</ parameters.loud >>
    steps:
      - checkout
`)

	for _, want := range []string{
		"10:23 keyword <<# parameters.loud >>",
		"10:49 keyword <</ parameters.loud >>",
		"10:71 keyword <<^ parameters.loud >>",
		"10:97 keyword <</ parameters.loud >>",
	} {
		assert.Check(t, cmp.Contains(tokens, want))
	}
}
