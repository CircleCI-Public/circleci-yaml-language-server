package main

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestParseContextsOrder(t *testing.T) {
	// Two workflows give build two contexts, in whichever order the
	// workflows map ranges.
	content := []byte(`version: 2.1
jobs:
  build:
    machine: true
    steps: [checkout]
workflows:
  a:
    jobs:
      - build:
          context: first
  b:
    jobs:
      - build:
          context: second
`)
	for range 50 {
		assert.Check(t, cmp.Len(checkParse("contexts.yml", content), 0))
	}
}
