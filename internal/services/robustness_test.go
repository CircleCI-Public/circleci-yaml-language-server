package languageservice

import (
	"flag"
	"fmt"
	"runtime/debug"
	"slices"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/dockerhub"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

// malformedConfigs are configs as they stand partway through being written:
// keys with no value yet, values of the wrong shape, dangling anchors and
// parameter references, and YAML that does not parse at all. A request on any
// of them, anywhere in them, must be answered rather than crash the server.
var malformedConfigs = map[string]string{
	"empty":                      "",
	"only version":               "version: 2.1\n",
	"version no val":             "version:",
	"bare scalar":                "hello",
	"bare list":                  "- a\n- b\n",
	"jobs scalar":                "version: 2.1\njobs: foo\n",
	"jobs list":                  "version: 2.1\njobs:\n  - build\n",
	"jobs empty":                 "version: 2.1\njobs:\n",
	"job empty":                  "version: 2.1\njobs:\n  build:\n",
	"job scalar":                 "version: 2.1\njobs:\n  build: 3\n",
	"job list":                   "version: 2.1\njobs:\n  build:\n    - a\n",
	"docker empty":               "version: 2.1\njobs:\n  build:\n    docker:\n",
	"docker scalar":              "version: 2.1\njobs:\n  build:\n    docker: cimg/base\n",
	"docker map":                 "version: 2.1\njobs:\n  build:\n    docker:\n      image: cimg/base\n",
	"docker dash":                "version: 2.1\njobs:\n  build:\n    docker:\n      -\n",
	"docker image empty":         "version: 2.1\njobs:\n  build:\n    docker:\n      - image:\n",
	"docker image quoted empty":  "version: 2.1\njobs:\n  build:\n    docker:\n      - image: \"\"\n",
	"docker image list":          "version: 2.1\njobs:\n  build:\n    docker:\n      - image: [a, b]\n",
	"docker image multiline":     "version: 2.1\njobs:\n  build:\n    docker:\n      - image: >\n          cimg/base\n",
	"docker image colon":         "version: 2.1\njobs:\n  build:\n    docker:\n      - image: :\n",
	"docker image at":            "version: 2.1\njobs:\n  build:\n    docker:\n      - image: foo@\n",
	"docker image slash":         "version: 2.1\njobs:\n  build:\n    docker:\n      - image: /\n",
	"docker unicode":             "version: 2.1\njobs:\n  build:\n    docker:\n      - image: \"éé/ü\"\n    steps: [checkout]\n",
	"machine empty":              "version: 2.1\njobs:\n  build:\n    machine:\n",
	"machine true":               "version: 2.1\njobs:\n  build:\n    machine: true\n",
	"machine image empty":        "version: 2.1\njobs:\n  build:\n    machine:\n      image:\n",
	"macos empty":                "version: 2.1\njobs:\n  build:\n    macos:\n",
	"macos xcode empty":          "version: 2.1\njobs:\n  build:\n    macos:\n      xcode:\n",
	"resource class empty":       "version: 2.1\njobs:\n  build:\n    docker:\n      - image: cimg/base:current\n    resource_class:\n",
	"executor ref empty":         "version: 2.1\njobs:\n  build:\n    executor:\n",
	"executor ref map empty":     "version: 2.1\njobs:\n  build:\n    executor:\n      name:\n",
	"executor ref list":          "version: 2.1\njobs:\n  build:\n    executor: [a]\n",
	"executors empty":            "version: 2.1\nexecutors:\n",
	"executor empty":             "version: 2.1\nexecutors:\n  e:\n",
	"executor scalar":            "version: 2.1\nexecutors:\n  e: foo\n",
	"steps empty":                "version: 2.1\njobs:\n  build:\n    steps:\n",
	"steps scalar":               "version: 2.1\njobs:\n  build:\n    steps: checkout\n",
	"steps dash":                 "version: 2.1\njobs:\n  build:\n    steps:\n      -\n",
	"steps run empty":            "version: 2.1\njobs:\n  build:\n    steps:\n      - run:\n",
	"steps run map empty":        "version: 2.1\njobs:\n  build:\n    steps:\n      - run:\n          command:\n",
	"steps nested list":          "version: 2.1\njobs:\n  build:\n    steps:\n      - - checkout\n",
	"steps when empty":           "version: 2.1\njobs:\n  build:\n    steps:\n      - when:\n",
	"steps when cond empty":      "version: 2.1\njobs:\n  build:\n    steps:\n      - when:\n          condition:\n          steps:\n",
	"steps unknown map":          "version: 2.1\njobs:\n  build:\n    steps:\n      - foo:\n          bar:\n",
	"step two keys":              "version: 2.1\njobs:\n  build:\n    steps:\n      - run: a\n        checkout:\n",
	"steps colon":                "version: 2.1\njobs:\n  build:\n    steps:\n      - :\n",
	"commands empty":             "version: 2.1\ncommands:\n",
	"command empty":              "version: 2.1\ncommands:\n  c:\n",
	"command steps empty":        "version: 2.1\ncommands:\n  c:\n    steps:\n",
	"command params empty":       "version: 2.1\ncommands:\n  c:\n    parameters:\n",
	"command param empty":        "version: 2.1\ncommands:\n  c:\n    parameters:\n      p:\n",
	"command param type empty":   "version: 2.1\ncommands:\n  c:\n    parameters:\n      p:\n        type:\n",
	"command param enum empty":   "version: 2.1\ncommands:\n  c:\n    parameters:\n      p:\n        type: enum\n        enum:\n",
	"command param default only": "version: 2.1\ncommands:\n  c:\n    parameters:\n      p:\n        default:\n",
	"parameters empty":           "version: 2.1\nparameters:\n",
	"parameter empty":            "version: 2.1\nparameters:\n  p:\n",
	"parameter scalar":           "version: 2.1\nparameters:\n  p: 3\n",
	"param ref open":             "version: 2.1\njobs:\n  build:\n    steps:\n      - run: echo << pipeline.parameters.\n",
	"param ref open2":            "version: 2.1\njobs:\n  build:\n    steps:\n      - run: echo <<\n",
	"param ref open3":            "version: 2.1\njobs:\n  build:\n    steps:\n      - run: echo << parameters\n",
	"param ref empty":            "version: 2.1\njobs:\n  build:\n    steps:\n      - run: echo << >>\n",
	"param ref close":            "version: 2.1\njobs:\n  build:\n    steps:\n      - run: echo >> <<\n",
	"param ref unicode":          "version: 2.1\njobs:\n  build:\n    parameters:\n      p:\n        type: string\n    steps:\n      - run: echo «é» << parameters.p >> é\n",
	"param ref as key":           "version: 2.1\njobs:\n  build:\n    << parameters.x >>:\n",
	"workflows empty":            "version: 2.1\nworkflows:\n",
	"workflow empty":             "version: 2.1\nworkflows:\n  main:\n",
	"workflow jobs empty":        "version: 2.1\nworkflows:\n  main:\n    jobs:\n",
	"workflow jobs dash":         "version: 2.1\nworkflows:\n  main:\n    jobs:\n      -\n",
	"workflow job empty map":     "version: 2.1\nworkflows:\n  main:\n    jobs:\n      - build:\n",
	"workflow requires empty":    "version: 2.1\njobs:\n  build:\n    docker: [{image: a}]\n    steps: [checkout]\nworkflows:\n  main:\n    jobs:\n      - build:\n          requires:\n",
	"workflow requires dash":     "version: 2.1\nworkflows:\n  main:\n    jobs:\n      - build:\n          requires:\n            -\n",
	"workflow requires map":      "version: 2.1\nworkflows:\n  main:\n    jobs:\n      - build:\n          requires:\n            - a: [success]\n            - b:\n",
	"workflow requires scalar":   "version: 2.1\nworkflows:\n  main:\n    jobs:\n      - build:\n          requires: a\n",
	"workflow filters empty":     "version: 2.1\nworkflows:\n  main:\n    jobs:\n      - build:\n          filters:\n            branches:\n",
	"workflow triggers empty":    "version: 2.1\nworkflows:\n  main:\n    triggers:\n      -\n",
	"workflow triggers sched":    "version: 2.1\nworkflows:\n  main:\n    triggers:\n      - schedule:\n",
	"workflow when empty":        "version: 2.1\nworkflows:\n  main:\n    when:\n",
	"workflow context empty":     "version: 2.1\nworkflows:\n  main:\n    jobs:\n      - build:\n          context:\n",
	"workflow matrix empty":      "version: 2.1\nworkflows:\n  main:\n    jobs:\n      - build:\n          matrix:\n            parameters:\n",
	"workflow matrix alias":      "version: 2.1\nworkflows:\n  main:\n    jobs:\n      - build:\n          matrix:\n            alias:\n",
	"workflow scalar":            "version: 2.1\nworkflows:\n  main: 3\n",
	"workflow version":           "version: 2.1\nworkflows:\n  version: 2\n",
	"job groups empty":           "version: 2.1\njob-groups:\n",
	"job group empty":            "version: 2.1\njob-groups:\n  g:\n",
	"job group jobs empty":       "version: 2.1\njob-groups:\n  g:\n    jobs:\n      -\n",
	"orbs empty":                 "version: 2.1\norbs:\n",
	"orb empty":                  "version: 2.1\norbs:\n  node:\n",
	"orb at":                     "version: 2.1\norbs:\n  node: circleci/node@\n",
	"orb slash":                  "version: 2.1\norbs:\n  node: circleci/\n",
	"orb only at":                "version: 2.1\norbs:\n  node: \"@\"\n",
	"orb only slash":             "version: 2.1\norbs:\n  node: /\n",
	"orb double at":              "version: 2.1\norbs:\n  node: a/b@1@2\n",
	"orb volatile":               "version: 2.1\norbs:\n  node: a/b@volatile\n",
	"orb list":                   "version: 2.1\norbs:\n  node: [a]\n",
	"orb use dangling":           "version: 2.1\norbs:\n  node: circleci/node@\njobs:\n  build:\n    steps:\n      - node/\n",
	"orb use slash":              "version: 2.1\njobs:\n  build:\n    steps:\n      - /\n",
	"orb local empty":            "version: 2.1\norbs:\n  local:\n    commands:\n",
	"orb local cmd empty":        "version: 2.1\norbs:\n  local:\n    commands:\n      c:\n",
	"orb local nested":           "version: 2.1\norbs:\n  local:\n    orbs:\n      inner:\n        commands:\n          c:\n            steps:\n",
	"orb local use":              "version: 2.1\norbs:\n  local:\n    commands:\n      c:\n        steps: [checkout]\njobs:\n  build:\n    steps:\n      - local/\n      - local/c:\n",
	"anchor dangling":            "version: 2.1\njobs:\n  build: &\n",
	"alias dangling":             "version: 2.1\njobs:\n  build: *\n",
	"alias unknown":              "version: 2.1\njobs:\n  build: *nope\n",
	"merge key empty":            "version: 2.1\njobs:\n  build:\n    <<:\n",
	"merge key alias":            "version: 2.1\ndefaults: &d\n  docker: [{image: a}]\njobs:\n  build:\n    <<: *d\n    <<: *\n",
	"anchor on key":              "version: 2.1\njobs:\n  &a build:\n",
	"anchor on image":            "version: 2.1\njobs:\n  build:\n    docker:\n      - image: &img\n",
	"alias on image":             "version: 2.1\nx: &img cimg/base\njobs:\n  build:\n    docker:\n      - image: *img\n",
	"tabs":                       "version: 2.1\njobs:\n\tbuild:\n\t\tsteps:\n",
	"crlf":                       "version: 2.1\r\njobs:\r\n  build:\r\n    steps:\r\n      - checkout\r\n",
	"unclosed quote":             "version: 2.1\njobs:\n  build:\n    steps:\n      - run: \"echo\n",
	"unclosed flow":              "version: 2.1\njobs:\n  build: {steps: [checkout\n",
	"flow everything":            "{version: 2.1, jobs: {build: {docker: [{image: cimg/base}], steps: [checkout, {run: echo}]}}, workflows: {main: {jobs: [build]}}}",
	"multidoc":                   "version: 2.1\n---\njobs:\n  build:\n---\n",
	"doc end":                    "version: 2.1\n...\njobs:\n",
	"directive":                  "%YAML 1.2\n---\nversion: 2.1\n",
	"tag":                        "version: 2.1\njobs: !!map\n  build: !foo\n",
	"deep indent":                "version: 2.1\njobs:\n  build:\n        steps:\n  - checkout\n",
	"dup keys":                   "version: 2.1\njobs:\n  build:\n    steps: [checkout]\n  build:\n    steps:\n",
	"null keys":                  "version: 2.1\n~: 3\nnull:\n",
	"setup empty":                "version: 2.1\nsetup:\n",
	"version weird":              "version: [2.1]\n",
	"emoji":                      "version: 2.1\njobs:\n  🚀:\n    steps:\n      - run: 🎉 << pipeline.git.branch >> 🎉\n",
	"only comments":              "# hello\n# world\n",
	"trailing space":             "version: 2.1\njobs:   \n  build:   \n    steps:   \n      -   ",
	"no newline":                 "version: 2.1\njobs:\n  build:\n    steps:\n      - run: echo",
	"key only":                   "version: 2.1\nj",
	"colon only":                 ":",
	"dash only":                  "-",
	"questionmark":               "? a\n: b\n",
	"block scalar":               "version: 2.1\njobs:\n  build:\n    steps:\n      - run: |\n",
	"serial group":               "version: 2.1\njobs:\n  build:\n    serial-group:\n",
	"job type":                   "version: 2.1\njobs:\n  build:\n    type:\n",
	"job type approval":          "version: 2.1\njobs:\n  build:\n    type: approval\n",
	"job params empty":           "version: 2.1\njobs:\n  build:\n    parameters:\n      p:\n        type: executor\n        default:\n",
	"job params steps":           "version: 2.1\njobs:\n  build:\n    parameters:\n      p:\n        type: steps\n    steps:\n      - steps: << parameters.p >>\n",
	"environment list":           "version: 2.1\njobs:\n  build:\n    environment:\n      - A=1\n",
	"environment empty":          "version: 2.1\njobs:\n  build:\n    environment:\n",
	"store artifacts empty":      "version: 2.1\njobs:\n  build:\n    steps:\n      - store_artifacts:\n",
	"save cache empty":           "version: 2.1\njobs:\n  build:\n    steps:\n      - save_cache:\n          paths:\n",
	"restore cache keys":         "version: 2.1\njobs:\n  build:\n    steps:\n      - restore_cache:\n          keys:\n            -\n",
	"checkout method":            "version: 2.1\njobs:\n  build:\n    steps:\n      - checkout:\n          method:\n",
	"merge self":                 "version: 2.1\njobs:\n  build: &a\n    <<: *a\n    docker: [{image: cimg/base:current}]\n",
	"merge cycle":                "version: 2.1\nx: &a\n  <<: *b\ny: &b\n  <<: *a\njobs:\n  build:\n    <<: *a\n",
	"step alias self":            "version: 2.1\njobs:\n  build:\n    steps:\n      - &s\n        when:\n          condition: true\n          steps: [*s]\n",
	"step alias self, block":     "version: 2.1\njobs:\n  build:\n    steps:\n      - &s\n        when:\n          condition: true\n          steps:\n            - *s\n",
	"step alias later":           "version: 2.1\njobs:\n  build:\n    steps:\n      - *s\n      - &s run: echo\n",
	"executor alias self":        "version: 2.1\nexecutors:\n  e: &e\n    <<: *e\n",
	"workflow job alias self":    "version: 2.1\nworkflows:\n  main:\n    jobs:\n      - &j build:\n          requires: [*j]\n",
}

// typedConfig is a whole config reaching most of what the handlers read,
// which TestRequestsSurviveTypingAConfig types out one byte at a time.
const typedConfig = `version: 2.1

orbs:
  local:
    commands:
      greet:
        parameters:
          to:
            type: string
            default: world
        steps:
          - run: echo hello << parameters.to >>

parameters:
  greeting:
    type: string
    default: hello

defaults: &defaults
  docker:
    - image: cimg/node:22.1.0
  resource_class: large

executors:
  linux:
    machine:
      image: ubuntu-2404:current

commands:
  setup:
    steps:
      - checkout
      - restore_cache:
          keys:
            - v1-{{ checksum "package-lock.json" }}

jobs:
  build:
    <<: *defaults
    parameters:
      target:
        type: enum
        enum: [a, b]
        default: a
    steps:
      - setup
      - local/greet:
          to: << parameters.target >>
      - run: echo << pipeline.parameters.greeting >>
      - when:
          condition: << pipeline.git.branch >>
          steps:
            - run: echo branch
  test:
    executor: linux
    steps:
      - checkout

workflows:
  main:
    jobs:
      - build:
          matrix:
            parameters:
              target: [a, b]
      - test:
          requires:
            - build
          filters:
            branches:
              only: main
`

// hermeticSettings are settings whose CircleCI and Docker Hub are fakes, with
// the temporary directory fetched orb sources are written under moved
// somewhere of the test's own, so that nothing a config names reaches a real
// service or outlives the test.
func hermeticSettings(t testing.TB) *session.Settings {
	t.Helper()

	t.Setenv("TMPDIR", t.TempDir())

	circleci := fakes.NewCircleCI(t)

	hub := fakes.NewDockerHub(t)
	hub.AddRepository("cimg", "node")
	hub.AddRepository("cimg", "base")
	hub.AddTag("cimg", "node", "22.1.0", "active")
	hub.AddTag("cimg", "node", "current", "active")

	settings := testHelpers.SettingsForHost(circleci.URL())
	settings.DockerHub = dockerhub.Config{BaseURL: hub.URL()}

	return settings
}

// survives runs a request, failing t rather than the whole test binary if it
// panics, so that one broken request does not hide the rest.
func survives(t testing.TB, request string, content string, pos protocol.Position, run func()) {
	t.Helper()

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Errorf("%s at %d:%d panicked: %v\ncontent: %q\n%s", request, pos.Line, pos.Character, recovered, content, debug.Stack())
		}
	}()

	run()
}

// positionRequest is a request the server answers about a position in a
// document.
type positionRequest string

const (
	askCompletion positionRequest = "completion"
	askHover      positionRequest = "hover"
	askDefinition positionRequest = "definition"
	askReferences positionRequest = "references"
)

var allPositionRequests = []positionRequest{askCompletion, askHover, askDefinition, askReferences}

// everyRequest makes each request the server answers on content: those about
// the whole document once, and each of requests at each of positions.
func everyRequest(t testing.TB, settings *session.Settings, content string, positions []protocol.Position, requests []positionRequest) {
	t.Helper()

	docURI := uri.File("/workspace/.circleci/config.yml")
	c := cache.New()
	c.FileCache.SetFile(cache.File{TextDocument: protocol.TextDocumentItem{URI: docURI, Text: content}})
	document := protocol.TextDocumentIdentifier{URI: docURI}

	var start protocol.Position
	survives(t, "diagnostics", content, start, func() {
		_, _ = DiagnosticFile(docURI, c, settings, "")
	})
	survives(t, "document symbols", content, start, func() {
		_, _ = DocumentSymbols(protocol.DocumentSymbolParams{TextDocument: document}, c, settings)
	})
	survives(t, "semantic tokens", content, start, func() {
		_ = SemanticTokens(protocol.SemanticTokensParams{TextDocument: document}, c, settings)
	})

	for _, pos := range positions {
		at := protocol.TextDocumentPositionParams{TextDocument: document, Position: pos}

		for _, request := range requests {
			survives(t, string(request), content, pos, func() {
				switch request {
				case askCompletion:
					_, _ = Complete(protocol.CompletionParams{TextDocumentPositionParams: at}, c, settings)
				case askHover:
					_, _ = Hover(protocol.HoverParams{TextDocumentPositionParams: at}, c, settings)
				case askDefinition:
					_, _ = Definition(protocol.DefinitionParams{TextDocumentPositionParams: at}, c, settings)
				case askReferences:
					_, _ = References(protocol.ReferenceParams{TextDocumentPositionParams: at}, c, settings)
				}
			})
		}
	}
}

// cursorPositions are the positions on each line of content where a cursor
// most often is: the start of its text, partway through it, and its end; and
// a couple past the end of the document, which a client can send for a
// document that has since changed.
func cursorPositions(content string) []protocol.Position {
	lines := strings.Split(content, "\n")

	var positions []protocol.Position
	for line, text := range lines {
		indent := len(text) - len(strings.TrimLeft(text, " "))
		for _, character := range slices.Compact([]int{indent, (indent + len(text)) / 2, len(text)}) {
			positions = append(positions, protocol.Position{Line: uint32(line), Character: uint32(character)})
		}
	}

	return append(positions,
		protocol.Position{Line: uint32(len(lines)), Character: 0},
		protocol.Position{Line: uint32(len(lines) + 5), Character: 3},
	)
}

// typingStops are where, typing content out, a request is most likely to find
// it in a state nothing else does: as each token is finished. A token is a run
// of word characters, a run of spaces, or a single other character; finishing
// a run of spaces is left out, as it is mostly indenting the next line.
func typingStops(content string) []int {
	kind := func(c byte) int {
		switch {
		case c == '_' || c == '-' || c == '.' || c == '/' ||
			'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9':
			return 1
		case c == ' ':
			return 2
		default:
			return 3
		}
	}

	stops := []int{0}
	for i := 1; i < len(content); i++ {
		finished := kind(content[i]) != kind(content[i-1]) || kind(content[i]) == 3
		if finished && kind(content[i-1]) != 2 {
			stops = append(stops, i)
		}
	}
	return append(stops, len(content))
}

// endOf is the position at the end of content, where the cursor is while
// content is being typed.
func endOf(content string) protocol.Position {
	lines := strings.Split(content, "\n")
	last := len(lines) - 1

	return protocol.Position{Line: uint32(last), Character: uint32(len(lines[last]))}
}

func TestRequestsSurviveMalformedConfigs(t *testing.T) {
	settings := hermeticSettings(t)

	for name, content := range malformedConfigs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			everyRequest(t, settings, content, cursorPositions(content), allPositionRequests)
		})
	}
}

func TestRequestsSurviveTypingAConfig(t *testing.T) {
	settings := hermeticSettings(t)

	for _, end := range typingStops(typedConfig) {
		t.Run(fmt.Sprint(end), func(t *testing.T) {
			t.Parallel()

			// What is asked about the position being typed at is completion.
			// The other requests are made at positions throughout the
			// malformed configs.
			typed := typedConfig[:end]
			everyRequest(t, settings, typed, []protocol.Position{endOf(typed)}, []positionRequest{askCompletion})
		})
	}
}

func TestRequestsSurviveCuttingALineShort(t *testing.T) {
	settings := hermeticSettings(t)
	lines := strings.Split(typedConfig, "\n")

	for index, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		t.Run(fmt.Sprint("line ", index), func(t *testing.T) {
			t.Parallel()

			// Halfway through the line's text.
			indent := len(line) - len(strings.TrimLeft(line, " "))
			cut := indent + (len(line)-indent)/2
			edited := append(append(append([]string{}, lines[:index]...), line[:cut]), lines[index+1:]...)
			everyRequest(t, settings, strings.Join(edited, "\n"), []protocol.Position{{Line: uint32(index), Character: uint32(cut)}}, allPositionRequests)
		})
	}
}

func FuzzRequests(f *testing.F) {
	// Run as a test, the seeds are only replayed, and the malformed configs
	// are already swept by TestRequestsSurviveMalformedConfigs. They are only
	// worth their time as somewhere for the fuzzer to start from.
	if fuzzing := flag.Lookup("test.fuzz"); fuzzing != nil && fuzzing.Value.String() != "" {
		for _, content := range malformedConfigs {
			end := endOf(content)
			f.Add(content, end.Line, end.Character)
		}
	}
	f.Add(typedConfig, uint32(0), uint32(0))

	settings := hermeticSettings(f)

	f.Fuzz(func(t *testing.T, content string, line, character uint32) {
		everyRequest(t, settings, content, []protocol.Position{{Line: line, Character: character}}, allPositionRequests)
	})
}
