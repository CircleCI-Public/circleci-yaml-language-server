package main

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/compile"
)

func TestReadFixture(t *testing.T) {
	const fixture = "a << x.y >>\n\nb\n=====\n{\"x.y\" \"1\", \"n\" -2 \"t\" true \"z\" nil}\n\na 1\n\nb\n-----\n{}\n\na \n\nb\n"

	text, cases, err := readFixture(fixture)
	assert.NilError(t, err)
	assert.Check(t, cmp.Equal(text, "a << x.y >>\n\nb"))
	assert.Assert(t, cmp.Len(cases, 2))
	assert.Check(t, cmp.DeepEqual(cases[0].env, compile.Env{"x.y": "1", "n": int64(-2), "t": true, "z": nil}))
	assert.Check(t, cmp.Equal(cases[0].want, "a 1\n\nb"))
	assert.Check(t, cmp.DeepEqual(cases[1].env, compile.Env{}))
	assert.Check(t, cmp.Equal(cases[1].want, "a \n\nb"))
}

func TestTemplates(t *testing.T) {
	dir := os.Getenv("REFERENCE_CORPUS_DIR")
	if dir == "" {
		t.Skip("REFERENCE_CORPUS_DIR is unset")
	}

	t.Run("replacer fixtures", func(t *testing.T) {
		rows, n, err := checkTemplates(filepath.Join(dir, "replacer"))
		assert.NilError(t, err)
		assert.Check(t, n != 0)
		for _, r := range rows {
			t.Errorf("%s: %s", r.File, r.Message)
		}
	})

	t.Run("workflow conditions", func(t *testing.T) {
		rows, n, err := checkWorkflowConditions(filepath.Join(dir, "logic"))
		assert.NilError(t, err)
		assert.Check(t, n != 0)
		for _, r := range rows {
			t.Errorf("%s:%d: %s", r.File, r.Line, r.Message)
		}
	})
}
