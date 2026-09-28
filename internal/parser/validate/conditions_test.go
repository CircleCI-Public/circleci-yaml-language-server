package validate

import (
	"slices"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
)

func TestExpressionConditionsAndFilters(t *testing.T) {
	CheckYamlErrors(t, []ValidateTestCase{
		{
			Name: "Conditions and filters that don't parse, at the spot",
			YamlContent: `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - checkout

workflows:
  main:
    when: pipeline.git.branch == "main" and and true
    jobs:
      - build:
          filters: "pipeline.git.branch starts-with"
  other:
    unless: pipeline.git.tag & true
    jobs:
      - build
`,
			OnlyErrors: true,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 11, Character: 44},
					End:   protocol.Position{Line: 11, Character: 47},
				}, `Invalid condition expression: Expected expression, found "and"`),
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 16, Character: 29},
					End:   protocol.Position{Line: 16, Character: 30},
				}, "Invalid condition expression: Unexpected character '&'"),
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 14, Character: 20},
					End:   protocol.Position{Line: 14, Character: 51},
				}, "Invalid filter expression: Expected expression, found end of expression"),
			},
		},
		{
			Name: "Expressions that parse, and values that aren't expressions",
			YamlContent: `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - checkout

workflows:
  main:
    when: pipeline.git.branch matches /^release\/.+$/ or not pipeline.parameters.skip
    jobs:
      - build:
          filters: pipeline.git.tag starts-with "v"
      - build:
          name: build-2
          filters: { branches: { only: main } }
  templated:
    when: << pipeline.parameters.run >>
    jobs:
      - build
  yaml-boolean:
    when: yes
    jobs:
      - build
  mapping:
    when: { equal: [ main, << pipeline.git.branch >> ] }
    jobs:
      - build

parameters:
  skip:
    type: boolean
    default: false
  run:
    type: boolean
    default: true
`,
			OnlyErrors:  true,
			Diagnostics: []protocol.Diagnostic{},
		},
	})
}

func TestStepConditionsUsingABareWord(t *testing.T) {
	val := CreateValidateFromYAML(`version: 2.1

commands:
  maybe:
    parameters:
      partition:
        type: string
        default: ""
    steps:
      - when:
          condition: always and pipeline.git.tag
          steps:
            - checkout
      - when:
          condition: parameters.partition starts-with "github/"
          steps:
            - checkout

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - maybe

workflows:
  main:
    jobs:
      - build
`)
	val.Validate()

	said := getDiagnosticMessages(val.Diagnostics)
	assert.Check(t, cmp.Contains(said, "Condition `always and pipeline.git.tag` is treated as always true. "+
		"Use a boolean (`true`/`false`), an expression, or a logic statement such as `equal:`."))
	alwaysTrue := slices.DeleteFunc(said, func(message string) bool {
		return !strings.Contains(message, "always true")
	})
	assert.Check(t, cmp.Len(alwaysTrue, 1), "only the bare word is warned about")
}
