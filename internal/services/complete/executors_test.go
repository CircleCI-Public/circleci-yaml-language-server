package complete

import (
	"slices"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

// dockerImage parses an executor with the given image line and returns the
// image, with the ranges the parser gave it.
func dockerImage(t *testing.T, imageLine string) ast.DockerImage {
	t.Helper()

	content := "version: 2.1\nexecutors:\n  e:\n    docker:\n" + imageLine + "\n"
	doc, err := yamlparser.ParseFromContent([]byte(content), testHelpers.DefaultSettings(), uri.File("/config.yml"), protocol.Position{})
	assert.NilError(t, err)
	t.Cleanup(doc.Close)

	executor, ok := doc.Executors["e"].(ast.DockerExecutor)
	assert.Assert(t, ok, "executor e is %T", doc.Executors["e"])
	assert.Assert(t, cmp.Len(executor.Image, 1))

	return executor.Image[0]
}

// at is the position of the cursor in the image line, which is line 4.
func at(character uint32) protocol.Position {
	return protocol.Position{Line: 4, Character: character}
}

func TestTypedImage(t *testing.T) {
	//            0         1         2         3
	//            0123456789012345678901234567890
	const line = "      - image: cimg/node:17.2.0"

	t.Run("is what precedes the cursor in the value", func(t *testing.T) {
		img := dockerImage(t, line)

		for character, want := range map[uint32]string{
			16: "c",
			21: "cimg/n",
			25: "cimg/node:",
			31: "cimg/node:17.2.0",
		} {
			typed, ok := typedImage(img, at(character))
			assert.Check(t, ok, "character %d", character)
			assert.Check(t, cmp.Equal(typed, want), "character %d", character)
		}
	})

	t.Run("is nothing outside the value", func(t *testing.T) {
		img := dockerImage(t, line)

		// The key, the colon and the space after it are all inside the
		// entry's range, which is where completion used to slice out of
		// bounds and take the server down.
		for character := uint32(0); character <= 15; character++ {
			typed, ok := typedImage(img, at(character))
			assert.Check(t, !ok, "character %d gave %q", character, typed)
		}

		typed, ok := typedImage(img, protocol.Position{Line: 5, Character: 16})
		assert.Check(t, !ok, "another line gave %q", typed)
	})

	t.Run("is nothing past the end of the value", func(t *testing.T) {
		img := dockerImage(t, line)

		typed, ok := typedImage(img, at(40))
		assert.Check(t, !ok, "gave %q", typed)
	})

	t.Run("lines up with a quoted value", func(t *testing.T) {
		img := dockerImage(t, `      - image: "cimg/node:17.2.0"`)

		typed, ok := typedImage(img, at(16))
		assert.Check(t, !ok, "the opening quote gave %q", typed)

		typed, ok = typedImage(img, at(22))
		assert.Check(t, ok)
		assert.Check(t, cmp.Equal(typed, "cimg/n"))
	})

	t.Run("lines up with an anchored value", func(t *testing.T) {
		img := dockerImage(t, `      - image: &node cimg/node:17.2.0`)

		typed, ok := typedImage(img, at(20))
		assert.Check(t, !ok, "the anchor gave %q", typed)

		typed, ok = typedImage(img, at(27))
		assert.Check(t, ok)
		assert.Check(t, cmp.Equal(typed, "cimg/n"))
	})
}

func TestCompleteExecutorMapping(t *testing.T) {
	const config = `version: 2.1

orbs:
  tools:
    executors:
      node:
        parameters:
          tag:
            type: string
        docker:
          - image: cimg/node:<< parameters.tag >>

executors:
  base:
    parameters:
      tag:
        type: string
      size:
        type: string
        default: medium
    docker:
      - image: cimg/base:<< parameters.tag >>
    resource_class: << parameters.size >>

jobs:
  local:
    executor:
      name: base
      tag: stable
      
    steps:
      - checkout
  from-orb:
    executor:
      name: tools/node
      
    steps:
      - checkout
  unnamed:
    executor:
      
    steps:
      - checkout
  by-name:
    executor: base
      
`
	at := func(text string) []string {
		return completionLabels(t, config, positionBelow(t, config, text, 6))
	}

	t.Run("a local executor is offered the parameters it isn't given", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(at("tag: stable"), []string{"size"}))
	})

	t.Run("an orb executor is offered its parameters", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(at("name: tools/node"), []string{"tag"}))
	})

	t.Run("an executor with no name yet is offered name", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(at("executor:"), []string{"name"}))
	})

	t.Run("an executor given by name is offered no keys", func(t *testing.T) {
		labels := at("executor: base")
		assert.Check(t, !slices.Contains(labels, "name"), "%q", labels)
		assert.Check(t, !slices.Contains(labels, "size"), "%q", labels)
	})
}

func TestCompleteExecutorKeys(t *testing.T) {
	const config = `version: 2.1

executors:
  node:
    docker:
      - image: cimg/node:lts
    resource_class: large
    
  mac:
    macos:
      xcode: 16.0.0
    
`
	t.Run("an executor is offered the keys it doesn't have besides its type", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(completionLabels(t, config, positionBelow(t, config, "resource_class: large", 4)), []string{
			"environment", "shell", "working_directory", "parameters", "description",
		}))
	})

	t.Run("whatever its type", func(t *testing.T) {
		assert.Check(t, cmp.DeepEqual(completionLabels(t, config, positionBelow(t, config, "xcode: 16.0.0", 4)), []string{
			"resource_class", "environment", "shell", "working_directory", "parameters", "description",
		}))
	})
}
