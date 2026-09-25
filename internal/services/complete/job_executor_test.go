package complete

import (
	"slices"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestCompleteJobExecutor(t *testing.T) {
	const config = `version: 2.1

executors:
  mac:
    macos:
      xcode: 16.0.0

jobs:
  on-docker:
    docker:
      - image: cimg/base:stable
    resource_class: 
    steps:
      - checkout
  on-machine:
    machine:
      image: ubuntu-2404:cu
      
    resource_class: me
    steps:
      - checkout
  on-mac:
    macos:
      xcode: 1
    steps:
      - checkout
  on-named:
    executor: mac
    resource_class: m4
    steps:
      - checkout
`
	fake := fakes.NewCircleCI(t)
	fake.SetMachineOfferings(fakes.MachineOfferings{
		Linux: map[string][]string{"medium": {"ubuntu-2404:current"}, "large": {"ubuntu-2404:current", "ubuntu-2204:current"}},
		MacOS: map[string][]string{"m4pro.medium": {"xcode:16.0.0", "xcode:26.0.0"}},
	})
	settings := testHelpers.SettingsForHost(fake.URL())
	labels := func(pos protocol.Position) []string {
		got := completionLabelsWith(t, settings, cache.New(), config, pos)
		slices.Sort(got)
		return got
	}
	endOf := func(text string) protocol.Position {
		pos := positionBelow(t, config, text, 0)
		pos.Line--
		pos.Character = uint32(len(configLine(config, pos.Line)))
		return pos
	}

	t.Run("a docker job's resource class is offered docker's", func(t *testing.T) {
		got := labels(endOf("resource_class:"))
		assert.Check(t, cmp.Contains(got, "medium+"))
		assert.Check(t, !slices.Contains(got, "m4pro.medium"), "macOS class offered to a docker job")
	})

	t.Run("a machine job's resource class is offered the machine's", func(t *testing.T) {
		got := labels(endOf("resource_class: me"))
		assert.Check(t, cmp.DeepEqual(got, []string{"large", "medium"}))
	})

	t.Run("a job naming an executor is offered the classes of its type", func(t *testing.T) {
		got := labels(endOf("resource_class: m4"))
		assert.Check(t, cmp.DeepEqual(got, []string{"m4pro.medium"}))
	})

	t.Run("a machine's image is offered the machine images", func(t *testing.T) {
		got := labels(endOf("image: ubuntu-2404:cu"))
		assert.Check(t, cmp.DeepEqual(got, []string{"ubuntu-2204:current", "ubuntu-2404:current"}))
	})

	t.Run("a machine is offered the keys it doesn't have", func(t *testing.T) {
		got := labels(positionBelow(t, config, "image: ubuntu-2404:cu", 6))
		assert.Check(t, cmp.DeepEqual(got, []string{"docker_layer_caching"}))
	})

	t.Run("a macOS job's xcode is offered the Xcode versions", func(t *testing.T) {
		got := labels(endOf("xcode: 1"))
		assert.Check(t, cmp.DeepEqual(got, []string{"16.0.0", "26.0.0"}))
	})
}

// configLine is a line of a config.
func configLine(config string, line uint32) string {
	return strings.Split(config, "\n")[line]
}
