package validate

import (
	"testing"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
)

func TestTemplates(t *testing.T) {
	CheckYamlErrors(t, []ValidateTestCase{
		{
			Name: "An unescaped heredoc, and an expression that doesn't parse",
			YamlContent: `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - run: |
          cat <<EOF > notes.txt
          done
          EOF
      - run: echo << pipeline.git.branch == "main" and and true >>

workflows:
  main:
    jobs:
      - build
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 8, Character: 14},
					End:   protocol.Position{Line: 8, Character: 16},
				}, "Unclosed '<<' tag ('<<' must be escaped as '\\<<')"),
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 11, Character: 55},
					End:   protocol.Position{Line: 11, Character: 58},
				}, `Expected expression, found "and"`),
			},
		},
		{
			Name: "Escaped tags, merge keys and sections are fine",
			YamlContent: `version: 2.1

defaults: &defaults
  docker:
    - image: cimg/base:current

commands:
  greet:
    parameters:
      loud:
        type: boolean
        default: false
    steps:
      - run: echo <<# parameters.loud >>HELLO<</ parameters.loud >><<^ parameters.loud >>hello<</ parameters.loud >>

jobs:
  build:
    <<: *defaults
    steps:
      - greet
      - run: cat \<<EOF > notes.txt

workflows:
  main:
    jobs:
      - build
`,
			OnlyErrors:  true,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name: "Version 2 configs aren't templates",
			YamlContent: `version: 2

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - run: cat <<EOF > notes.txt

workflows:
  version: 2
  main:
    jobs:
      - build
`,
			OnlyErrors:  true,
			Diagnostics: []protocol.Diagnostic{},
		},
	})
}
