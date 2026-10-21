package parser_test

import (
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestMergeKeySequence(t *testing.T) {
	content := []byte(`version: 2.1
jobs:
  small: &small
    resource_class: small
    shell: /bin/sh
    docker:
      - image: cimg/base:current
    steps: [checkout]
  large: &large
    resource_class: large
    shell: /bin/bash
    working_directory: /large
    steps: [checkout]
  first-wins:
    <<: [*small, *large]
  own-key-wins:
    <<: [*small, *large]
    resource_class: medium
  missing-anchor-skipped:
    <<: [*missing, *large]
`)

	// Map iteration order varies between runs, so one parse could pass by
	// luck.
	for range 50 {
		doc, err := parser.ParseFromContent(content, testHelpers.DefaultSettings(), uri.File(""), protocol.Position{})
		assert.NilError(t, err)

		t.Run("the first anchor to set a key wins", func(t *testing.T) {
			job := doc.Jobs["first-wins"]
			assert.Check(t, cmp.Equal(job.ResourceClass, "small"))
			assert.Check(t, cmp.Equal(job.Shell, "/bin/sh"))
			assert.Check(t, cmp.Equal(job.WorkingDirectory, "/large"), "keys only a later anchor sets still merge")
		})

		t.Run("the mapping's own key wins", func(t *testing.T) {
			assert.Check(t, cmp.Equal(doc.Jobs["own-key-wins"].ResourceClass, "medium"))
		})

		t.Run("an undefined anchor doesn't stop the rest merging", func(t *testing.T) {
			assert.Check(t, cmp.Equal(doc.Jobs["missing-anchor-skipped"].ResourceClass, "large"))
		})

		doc.Close()
	}
}
