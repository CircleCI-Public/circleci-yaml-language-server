package main

import (
	"fmt"
	"os"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

// knownGaps are the problems parse finds in this repo's configs that the
// parser has yet to fix, so that fixing one fails until it's removed here.
var knownGaps = map[string][]string{
	"internal/parser/testdata/conditional-steps/as-args/build.yml": {
		"30: 1 when step(s) not in CompileInfo",
		"34: 1 when step(s) not in CompileInfo",
	},
}

func TestParse(t *testing.T) {
	t.Run("this repo's configs", func(t *testing.T) {
		configs, err := repoConfigs("../../..")
		assert.NilError(t, err)
		assert.Check(t, len(configs) != 0)
		checkAll(t, configs)
	})

	t.Run("the reference corpus", func(t *testing.T) {
		dir := os.Getenv("REFERENCE_CORPUS_DIR")
		if dir == "" {
			t.Skip("REFERENCE_CORPUS_DIR is unset")
		}
		configs, err := referenceConfigs(dir)
		assert.NilError(t, err)
		assert.Check(t, len(configs) != 0)
		checkAll(t, configs)
	})
}

func checkAll(t *testing.T, configs []config) {
	for _, c := range configs {
		t.Run(c.name, func(t *testing.T) {
			content, err := os.ReadFile(c.path())
			assert.NilError(t, err)

			var problems []string
			for _, r := range checkParse(c.name, content) {
				problems = append(problems, fmt.Sprintf("%d: %s", r.Line, r.Message))
			}
			assert.Check(t, cmp.DeepEqual(problems, knownGaps[c.name]))
		})
	}
}
