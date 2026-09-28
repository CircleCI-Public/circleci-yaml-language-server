package acceptance

import (
	"slices"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

// dockerConfig names an image Docker Hub has, one it doesn't, and a tag of the
// first that it doesn't.
const dockerConfig = `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/node:22.1.0
    steps:
      - run: echo building
  lint:
    docker:
      - image: cimg/nope:1.0
    steps:
      - run: echo linting
  deploy:
    docker:
      - image: cimg/node:99.0.0
    steps:
      - run: echo deploying

workflows:
  main:
    jobs:
      - build
      - lint
      - deploy
`

// withCimgNode gives a session's Docker Hub the cimg/node image, tagged
// 22.1.0.
func withCimgNode(session *session) {
	session.dockerHub.AddRepository("cimg", "node")
	session.dockerHub.AddTag("cimg", "node", "22.1.0", "active")
}

func TestDockerImageDiagnostics(t *testing.T) {
	session := start(t, linkedProjectFake(t), dockerConfig, testToken)
	withCimgNode(session)

	diagnostics := session.open(t, dockerConfig)

	t.Run("reports an image Docker Hub does not have", func(t *testing.T) {
		assert.Check(t, cmp.Contains(diagnostics, `Docker image not found "cimg/nope:1.0"`))
	})

	t.Run("reports a tag Docker Hub does not have", func(t *testing.T) {
		assert.Check(t, slices.ContainsFunc(diagnostics, func(d string) bool {
			return strings.Contains(d, `has no tag "99.0.0"`)
		}), "diagnostics: %q", diagnostics)
	})

	t.Run("reports nothing else", func(t *testing.T) {
		assert.Check(t, cmp.Len(diagnostics, 2))
	})
}

// imageConfig leaves an image name to be completed.
const imageConfig = `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/no
    steps:
      - run: echo building
`

func TestDockerImageCompletion(t *testing.T) {
	session := start(t, linkedProjectFake(t), imageConfig, testToken)
	withCimgNode(session)

	session.open(t, imageConfig)

	const prefix = "      - image: cimg/no"
	line := slices.Index(strings.Split(imageConfig, "\n"), prefix)
	assert.Assert(t, line >= 0, "the config must leave an image to complete")

	completions, err := session.client.Completion(session.workspace.URI(), position(uint32(line), uint32(len(prefix))))
	assert.NilError(t, err)
	assert.Assert(t, completions != nil)

	labels := make([]string, 0, len(completions.Items))
	for _, item := range completions.Items {
		labels = append(labels, item.Label)
	}
	assert.Check(t, cmp.Contains(labels, "cimg/node:latest"))
}
