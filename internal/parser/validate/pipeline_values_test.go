package validate

import (
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestPipelineValueWarnings(t *testing.T) {
	val := CreateValidateFromYAML(`version: 2.1

parameters:
  deploy:
    type: boolean
    default: false

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - run: echo << pipeline.git.brnach >> << pipeline.git.branch >> << pipeline.parameters.deploy >>
      - run: echo << pipeline.trigger_parameters.circleci.event_type >>

workflows:
  main:
    when: pipeline.git.tag != "" or pipeline.git.tagg != ""
    jobs:
      - build
`)
	val.Validate(t.Context())

	type found struct {
		Message    string
		Range      protocol.Range
		Deprecated bool
	}
	got := []found{}
	for _, d := range *val.Diagnostics {
		message := string(d.Message.(protocol.String))
		if strings.HasPrefix(message, "Unknown pipeline value") || strings.Contains(message, "is deprecated, use") {
			got = append(got, found{Message: message, Range: d.Range, Deprecated: len(d.Tags.Slice()) > 0})
		}
	}

	assert.Check(t, cmp.DeepEqual(got, []found{
		{
			Message: "Unknown pipeline value pipeline.git.brnach",
			Range:   protocol.Range{Start: protocol.Position{Line: 12, Character: 21}, End: protocol.Position{Line: 12, Character: 40}},
		},
		{
			Message:    "pipeline.trigger_parameters.circleci.event_type is deprecated, use pipeline.event.name instead",
			Range:      protocol.Range{Start: protocol.Position{Line: 13, Character: 21}, End: protocol.Position{Line: 13, Character: 68}},
			Deprecated: true,
		},
		{
			Message: "Unknown pipeline value pipeline.git.tagg",
			Range:   protocol.Range{Start: protocol.Position{Line: 17, Character: 36}, End: protocol.Position{Line: 17, Character: 53}},
		},
	}))
}
