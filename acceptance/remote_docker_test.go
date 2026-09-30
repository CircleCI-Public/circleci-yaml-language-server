package acceptance

import (
	"slices"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
)

// remoteDockerConfig asks for a Docker version the catalog offers in one job,
// and one it doesn't in another.
const remoteDockerConfig = `version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - setup_remote_docker:
          version: docker29
  publish:
    docker:
      - image: cimg/base:current
    resource_class: large
    steps:
      - setup_remote_docker:
          version: 24.0.9

workflows:
  main:
    jobs:
      - build
      - publish
`

// versionPrefix is where remoteDockerCompletionConfig leaves a version to be
// completed. Its trailing space is kept out of the raw string, where editors
// strip it.
const versionPrefix = "          version: "

const remoteDockerCompletionConfig = `version: 2.1

jobs:
  release:
    docker:
      - image: cimg/base:current
    steps:
      - setup_remote_docker:
` + versionPrefix + `
`

// remoteDockerFake has a catalog with the Docker executor's classes and the
// remote Docker versions of their machines.
func remoteDockerFake(t *testing.T) *fakes.CircleCI {
	fake := linkedProjectFake(t)
	fake.SetMachineOfferings(fakes.MachineOfferings{
		Linux:  map[string][]string{"medium": {"ubuntu-2404:current"}},
		Docker: map[string][]string{"medium": {}, "large": {}},
		RemoteDocker: map[string][]string{
			"medium": {"default", "docker29"},
			"large":  {"default", "docker29"},
		},
	})
	return fake
}

func TestRemoteDockerDiagnostics(t *testing.T) {
	session := start(t, remoteDockerFake(t), remoteDockerConfig, testToken)
	session.dockerHub.AddRepository("cimg", "base")
	session.dockerHub.AddTag("cimg", "base", "current", "active")

	diagnostics := session.open(t, remoteDockerConfig)

	assert.Check(t, cmp.DeepEqual(diagnostics, []string{
		`Docker version "24.0.9" isn't available to setup_remote_docker on resource class "large"`,
	}))
}

func TestRemoteDockerCompletion(t *testing.T) {
	session := start(t, remoteDockerFake(t), remoteDockerCompletionConfig, testToken)
	session.open(t, remoteDockerCompletionConfig)

	t.Run("completes the versions the catalog offers", func(t *testing.T) {
		line := slices.Index(strings.Split(remoteDockerCompletionConfig, "\n"), versionPrefix)
		assert.Assert(t, line >= 0, "the config must leave a version to complete")

		completions, err := session.client.Completion(session.workspace.URI(), position(uint32(line), uint32(len(versionPrefix))))
		assert.NilError(t, err)
		assert.Assert(t, completions != nil)

		labels := make([]string, 0, len(completions.Items))
		for _, item := range completions.Items {
			labels = append(labels, item.Label)
		}
		assert.Check(t, cmp.DeepEqual(labels, []string{"default", "docker29"}))
	})
}
