package hover

import (
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestJobInvocation(t *testing.T) {
	const config = `version: 2.1

jobs:
  deploy:
    description: Ship it.
    parameters:
      env:
        type: enum
        enum: [staging, prod]
    docker:
      - image: cimg/base:stable
    steps:
      - run: ./deploy << parameters.env >>

job-groups:
  release:
    jobs:
      - deploy:
          env: prod

workflows:
  main:
    jobs:
      - deploy:
          env: staging
      - hold:
          type: approval
`
	doc, err := yamlparser.ParseFromContent([]byte(config), testHelpers.DefaultSettings(), uri.File("/config.yml"), protocol.Position{})
	assert.NilError(t, err)
	t.Cleanup(doc.Close)

	const deploy = "**deploy** job\n\nShip it.\n\nParameters:\n\n- `env` (enum, required)"

	t.Run("a workflow's job shows the job's description and parameters", func(t *testing.T) {
		got, ok := JobInvocation(doc, cache.New(), protocol.Position{Line: 23, Character: 10})
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(got, deploy))
	})

	t.Run("so does a job group's", func(t *testing.T) {
		got, ok := JobInvocation(doc, cache.New(), protocol.Position{Line: 17, Character: 10})
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(got, deploy))
	})

	t.Run("an approval job, which names no job, has none", func(t *testing.T) {
		_, ok := JobInvocation(doc, cache.New(), protocol.Position{Line: 25, Character: 10})
		assert.Check(t, !ok)
	})
}
