package hover

import (
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestSchemaKey(t *testing.T) {
	const config = `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:stable
    resource_class: large
    steps:
      - checkout
      - run:
          command: make
      - run: {name: Test, command: make test}
      - deploy: ./deploy.sh
    "parallelism": 2

workflows:
  main:
    jobs:
      - build:
          requires: []
`
	doc, err := yamlparser.ParseFromContent([]byte(config), testHelpers.DefaultSettings(), uri.File("/config.yml"), protocol.Position{})
	assert.NilError(t, err)
	t.Cleanup(doc.Close)

	lines := strings.Split(config, "\n")
	at := func(line int, text string) protocol.Position {
		return protocol.Position{Line: uint32(line), Character: uint32(strings.Index(lines[line], text) + 1)}
	}

	described := []struct {
		name string
		at   protocol.Position
		want string
	}{
		{name: "a job's key", at: at(6, "resource_class"), want: "Resource class for the job."},
		{name: "a top-level key", at: at(2, "jobs"), want: "Jobs are collections of steps."},
		{name: "a step", at: at(9, "run"), want: "Used for invoking all command-line programs"},
		{name: "a step's option", at: at(10, "command"), want: "Command to run via the shell"},
		{name: "an option in a flow mapping", at: at(11, "command"), want: "Command to run via the shell"},
		{name: "the deprecated deploy step", at: at(12, "deploy"), want: "**Deprecated.** Replace it with `run`"},
		{name: "a quoted key", at: at(13, "parallelism"), want: "An integer, or an expression that evaluates to one"},
		{name: "a workflow job's option", at: at(19, "requires"), want: "you must explicitly require any dependencies"},
	}
	for _, tc := range described {
		t.Run(tc.name+" is described", func(t *testing.T) {
			got, ok := SchemaKey(doc, cache.New(), tc.at)
			assert.Assert(t, ok)
			assert.Check(t, cmp.Contains(got, tc.want))
		})
	}

	undescribed := []struct {
		name string
		at   protocol.Position
	}{
		{name: "a value", at: at(6, "large")},
		{name: "a job's name", at: at(3, "build")},
		{name: "a step written as a string", at: at(8, "checkout")},
		{name: "a job named in a workflow", at: at(18, "build")},
		{name: "a blank line", at: protocol.Position{Line: 14}},
	}
	for _, tc := range undescribed {
		t.Run(tc.name+" is not", func(t *testing.T) {
			_, ok := SchemaKey(doc, cache.New(), tc.at)
			assert.Check(t, !ok)
		})
	}
}
