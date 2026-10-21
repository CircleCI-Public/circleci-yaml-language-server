package acceptance

import (
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
)

// TestResourceClassesFromCatalog checks that a resource class is described
// as the catalog describes it, by name and size, in its hover and in
// completion.
func TestResourceClassesFromCatalog(t *testing.T) {
	fake := linkedProjectFake(t)
	fake.SetMachineOfferings(fakes.MachineOfferings{
		Linux:  map[string][]string{"large": {"ubuntu-2404:current"}},
		Docker: map[string][]string{"medium": {}, "large": {}},
		ResourceClasses: map[string]map[string]fakes.ResourceClass{
			"linux": {"large": {Name: "Linux Large", CPU: 4, RAMMB: 15360}},
			"docker": {
				"medium": {Name: "Medium", CPU: 2, RAMMB: 4096},
				"large":  {Name: "Large", CPU: 4, RAMMB: 8192},
			},
		},
	})

	session := start(t, fake, resourceClassConfig, testToken)
	session.open(t, resourceClassConfig)

	t.Run("hovering the class says what it is", func(t *testing.T) {
		hover, err := session.client.Hover(session.workspace.URI(), position(6, 22))
		assert.NilError(t, err)
		assert.Assert(t, hover != nil, "no hover")
		markup, ok := hover.Contents.(*protocol.MarkupContent)
		assert.Assert(t, ok, "hover contents are %T, not markup", hover.Contents)
		assert.Check(t, cmp.Equal(markup.Value, "**large** resource class\n\nLarge: 4 vCPUs, 8 GB RAM"))
	})

	t.Run("completion names and sizes the Docker executor's classes", func(t *testing.T) {
		list, err := session.client.Completion(session.workspace.URI(), position(6, 25))
		assert.NilError(t, err)
		assert.Assert(t, list != nil, "no completion")
		details := map[string]string{}
		for _, item := range list.Items {
			details[item.Label], _ = item.Detail.Get()
		}
		assert.Check(t, cmp.DeepEqual(details, map[string]string{
			"medium": "Medium: 2 vCPUs, 4 GB RAM",
			"large":  "Large: 4 vCPUs, 8 GB RAM",
		}))
	})
}

// xcodeConfig has an executor whose Xcode version is on line 5, and a job
// with a deprecated one.
const xcodeConfig = `version: 2.1

executors:
  mac:
    macos:
      xcode: "16.4"
    resource_class: m4pro.medium

jobs:
  build:
    macos:
      xcode: 26.1.1
    resource_class: m4pro.medium
    steps:
      - checkout
  package:
    executor: mac
    steps:
      - checkout

workflows:
  main:
    jobs:
      - build
      - package
`

// TestXcodeVersionsFromCatalog checks that the Xcode versions the catalog
// lists are offered, and the ones it deprecates are reported.
func TestXcodeVersionsFromCatalog(t *testing.T) {
	fake := linkedProjectFake(t)
	fake.SetMachineOfferings(fakes.MachineOfferings{
		Linux:      map[string][]string{"medium": {"ubuntu-2404:current"}},
		MacOS:      map[string][]string{"m4pro.medium": {"xcode:16.4", "xcode:16.4.0", "xcode:26.0.1"}},
		Deprecated: map[string][]string{"macos": {"xcode:26.1.1"}},
	})

	session := start(t, fake, xcodeConfig, testToken)
	diagnostics := session.open(t, xcodeConfig)

	t.Run("a deprecated version is reported", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(diagnostics, []string{`Xcode version "26.1.1" is deprecated`}))
	})

	t.Run("an executor's version is completed to the catalog's, replacing what is written", func(t *testing.T) {
		end := position(5, uint32(len(`      xcode: "16.4`)))
		list, err := session.client.Completion(session.workspace.URI(), end)
		assert.NilError(t, err)
		assert.Assert(t, list != nil, "no completion")

		written := protocol.Range{Start: position(5, uint32(len(`      xcode: "`))), End: end}
		got := []protocol.TextEdit{}
		for _, item := range list.Items {
			edit, ok := item.TextEdit.(*protocol.TextEdit)
			if assert.Check(t, ok, "item %q has no text edit", item.Label) {
				got = append(got, *edit)
			}
		}
		assert.Check(t, cmp.DeepEqual(got, []protocol.TextEdit{
			{Range: written, NewText: "26.0.1"},
			{Range: written, NewText: "16.4.0"},
			{Range: written, NewText: "16.4"},
		}))
	})
}
