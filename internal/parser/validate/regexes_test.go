package validate

import (
	"testing"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
)

func TestRegexes(t *testing.T) {
	CheckYamlErrors(t, []ValidateTestCase{
		{
			Name: "Patterns the compiler can't compile, in a logic statement and a filter",
			YamlContent: `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - when:
          condition:
            and:
              - false
              - matches:
                  pattern: "(.+)\\1"
                  value: << pipeline.git.branch >>
          steps:
            - checkout

workflows:
  main:
    when:
      matches: {pattern: "[", value: << pipeline.git.branch >>}
    jobs:
      - build:
          filters:
            branches:
              only: [main, "/^(?!main).*$/"]
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 12, Character: 27},
					End:   protocol.Position{Line: 12, Character: 36},
				}, `Invalid regular expression: (.+)\1`),
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 20, Character: 25},
					End:   protocol.Position{Line: 20, Character: 28},
				}, "Invalid regular expression: ["),
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 25, Character: 27},
					End:   protocol.Position{Line: 25, Character: 43},
				}, "Invalid regular expression: ^(?!main).*$"),
			},
		},
		{
			Name: "Patterns it can, and a filter that isn't a pattern",
			YamlContent: `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - checkout

workflows:
  main:
    when:
      matches:
        pattern: "^release/.+$"
        value: << pipeline.git.branch >>
    jobs:
      - build:
          filters:
            branches:
              only: /release\/.*/
              ignore: "(not a pattern"
            tags:
              only: /^v\d+/
`,
			OnlyErrors:  true,
			Diagnostics: []protocol.Diagnostic{},
		},
	})
}
