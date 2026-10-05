package hover

import (
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestResourceClass(t *testing.T) {
	const config = `version: 2.1

executors:
  big:
    machine:
      image: ubuntu-2404:current
    resource_class: large
  mapped:
    machine:
      image: ubuntu-2404:current
      resource_class: arm.medium

jobs:
  docker-job:
    docker:
      - image: cimg/base:current
    resource_class: medium
    steps:
      - setup_remote_docker:
          resource_class: large
  machine-job:
    machine:
      image: ubuntu-2404:current
    resource_class: medium
    steps:
      - checkout
  mac-job:
    macos:
      xcode: 26.5.0
    resource_class: m4pro.medium
    steps:
      - checkout
  named-job:
    executor: big
    resource_class: xlarge
    steps:
      - checkout
  runner-job:
    machine: true
    resource_class: acme/runner
    steps:
      - checkout
`
	doc, err := yamlparser.ParseFromContent([]byte(config), testHelpers.DefaultSettings(), uri.File("/config.yml"), protocol.Position{})
	assert.NilError(t, err)
	t.Cleanup(doc.Close)

	c := cache.New()
	offerings := testHelpers.MachineOfferings()
	offerings.ResourceClasses = map[string]map[string]circleci.ResourceClass{
		circleci.ExecutorLinux: {
			"medium":     {Name: "Linux Medium", CPU: 2, RAMMB: 8192},
			"large":      {Name: "Linux Large", CPU: 4, RAMMB: 15360},
			"xlarge":     {Name: "Linux X-Large", CPU: 8, RAMMB: 32768},
			"arm.medium": {Name: "Arm Linux Medium", CPU: 2, RAMMB: 8192},
		},
		circleci.ExecutorDocker: {"medium": {Name: "Medium", CPU: 2, RAMMB: 4096}},
		circleci.ExecutorMacOS:  {"m4pro.medium": {Name: "M4 Pro Medium", CPU: 6, RAMMB: 28672}},
	}
	c.MachineOfferingsCache.Set(offerings)

	// at is the position of the last character of the nth occurrence of text
	// in the config.
	at := func(text string, nth int) protocol.Position {
		t.Helper()
		lines := strings.Split(config, "\n")
		for i, line := range lines {
			if col := strings.Index(line, text); col >= 0 {
				if nth == 0 {
					return protocol.Position{Line: uint32(i), Character: uint32(col + len(text) - 1)}
				}
				nth--
			}
		}
		t.Fatalf("%q isn't in the config", text)
		return protocol.Position{}
	}
	hoverAt := func(pos protocol.Position) (string, bool) {
		return ResourceClass(t.Context(), doc, c, pos)
	}

	t.Run("a Docker job's class is the Docker executor's", func(t *testing.T) {
		got, ok := hoverAt(at("resource_class: medium", 0))
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(got, "**medium** resource class\n\nMedium: 2 vCPUs, 4 GB RAM"))
	})

	t.Run("a machine job's class of the same name is the machine's", func(t *testing.T) {
		got, ok := hoverAt(at("resource_class: medium", 1))
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(got, "**medium** resource class\n\nLinux Medium: 2 vCPUs, 8 GB RAM"))
	})

	t.Run("a macOS job's class", func(t *testing.T) {
		got, ok := hoverAt(at("resource_class: m4pro.medium", 0))
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(got, "**m4pro.medium** resource class\n\nM4 Pro Medium: 6 vCPUs, 28 GB RAM"))
	})

	t.Run("a named executor's class", func(t *testing.T) {
		got, ok := hoverAt(at("resource_class: large", 0))
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(got, "**large** resource class\n\nLinux Large: 4 vCPUs, 15 GB RAM"))
	})

	t.Run("a class given inside the machine map", func(t *testing.T) {
		got, ok := hoverAt(at("resource_class: arm.medium", 0))
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(got, "**arm.medium** resource class\n\nArm Linux Medium: 2 vCPUs, 8 GB RAM"))
	})

	t.Run("a job's class overriding the executor it names", func(t *testing.T) {
		got, ok := hoverAt(at("resource_class: xlarge", 0))
		assert.Assert(t, ok)
		assert.Check(t, cmp.Equal(got, "**xlarge** resource class\n\nLinux X-Large: 8 vCPUs, 32 GB RAM"))
	})

	t.Run("none for setup_remote_docker's, which is ignored", func(t *testing.T) {
		_, ok := hoverAt(at("resource_class: large", 1))
		assert.Check(t, !ok)
	})

	t.Run("none for a self-hosted runner's", func(t *testing.T) {
		_, ok := hoverAt(at("resource_class: acme/runner", 0))
		assert.Check(t, !ok)
	})

	t.Run("none on the key", func(t *testing.T) {
		_, ok := hoverAt(at("resource_class", 0))
		assert.Check(t, !ok)
	})
}
