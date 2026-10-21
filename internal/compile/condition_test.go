package compile

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestEvalCondition(t *testing.T) {
	env := Env{
		"pipeline.git.branch":   "main",
		"pipeline.git.tag":      nil,
		"pipeline.parameters.n": int64(0),
		"parameters.deploy":     true,
	}

	t.Run("truthiness", func(t *testing.T) {
		for _, tc := range []struct {
			src  string
			want bool
		}{
			{src: "false", want: false},
			{src: "0", want: false},
			{src: `""`, want: false},
			{src: "null", want: false},
			{src: "true", want: true},
			{src: "1", want: true},
			{src: `"   "`, want: true},
			{src: "[]", want: true},
			{src: "{}", want: true},
			{src: "{condition: false}", want: true},
			{src: "<< pipeline.parameters.n >>", want: false},
			{src: "<< pipeline.git.tag >>", want: false},
			{src: "<< pipeline.missing >>", want: false},
			{src: "<< parameters.deploy >>", want: true},
		} {
			got, err := EvalCondition(value(t, tc.src), env, At{})
			assert.Check(t, err, "%s", tc.src)
			assert.Check(t, cmp.Equal(got, tc.want), "%s", tc.src)
		}
	})

	t.Run("operators", func(t *testing.T) {
		for _, tc := range []struct {
			src  string
			want bool
		}{
			{src: "and: [true, 1, x]", want: true},
			{src: "and: [true, 0]", want: false},
			{src: "and: []", want: false},
			{src: "and:", want: false},
			{src: "or: [false, 0, x]", want: true},
			{src: "or: [false, '']", want: false},
			{src: "or: []", want: false},
			{src: "not: false", want: true},
			{src: "not: x", want: false},
			{src: "not:", want: true},
			{src: "equal: [main, << pipeline.git.branch >>]", want: true},
			{src: "equal: [main, dev]", want: false},
			{src: "equal: [1, '1']", want: false},
			{src: "equal: [[1, {a: b}], [1, {a: b}]]", want: true},
			{src: "equal: [x]", want: true},
			{src: "equal: []", want: false},
			{src: "equal: [true, {and: [1, x]}]", want: true},
			{src: "matches: {pattern: /^ma.n$/, value: << pipeline.git.branch >>}", want: true},
			{src: "matches: {pattern: ai, value: << pipeline.git.branch >>}", want: false},
			{src: "matches: {pattern: '', value: << pipeline.git.tag >>}", want: true},
			{src: "{or: [true], and: [false]}", want: true},
			{src: "and: [{not: {or: [false, {equal: [a, a]}]}}]", want: false},
		} {
			got, err := EvalCondition(value(t, tc.src), env, At{})
			assert.Check(t, err, "%s", tc.src)
			assert.Check(t, cmp.Equal(got, tc.want), "%s", tc.src)
		}
	})

	t.Run("errors", func(t *testing.T) {
		deep := strings.Repeat("{not: ", 101) + "true" + strings.Repeat("}", 101)
		for _, tc := range []struct {
			src, message string
		}{
			{src: "and: x", message: "`and` condition requires a list, got: \"x\""},
			{src: "or: {a: 1}", message: "`or` condition requires a list, got: {:a 1}"},
			{src: "equal: 1", message: "`equal` condition requires a list, got: 1"},
			{src: "matches: {pattern: '[', value: x}", message: "Invalid regular expression: ["},
			{src: "matches: {pattern: x, value: 1}", message: "Invalid regular expression value"},
			{src: deep, message: "Logic is nested too deeply"},
		} {
			_, err := EvalCondition(value(t, tc.src), env, At{})
			assert.Check(t, cmp.ErrorContains(err, tc.message), "%s", tc.src)
		}
	})
}

func TestEvalWorkflowWhen(t *testing.T) {
	env := Env{"pipeline.git.branch": "main", "pipeline.parameters.run": false}

	t.Run("a string with no variables is an expression", func(t *testing.T) {
		for _, tc := range []struct {
			src  string
			want bool
		}{
			{src: `pipeline.git.branch == "main"`, want: true},
			{src: `pipeline.git.branch == "dev"`, want: false},
			{src: `"a string"`, want: true},
			{src: "0", want: true},
		} {
			got, err := EvalWorkflowWhen(value(t, "'"+strings.ReplaceAll(tc.src, "'", "''")+"'"), env, At{})
			assert.Check(t, err, "%s", tc.src)
			assert.Check(t, cmp.Equal(got, tc.want), "%s", tc.src)
		}
	})

	t.Run("an unknown variable in an expression is an error", func(t *testing.T) {
		_, err := EvalWorkflowWhen(value(t, "pipeline.nope"), env, At{})
		assert.Check(t, cmp.ErrorContains(err, "Error evaluating workflow filter expression:\nReferred to a variable \"pipeline.nope\" that does not exist"))
	})

	t.Run("anything else is a logic statement", func(t *testing.T) {
		for src, want := range map[string]bool{
			"<< pipeline.parameters.run >>":            false,
			"<< pipeline.git.branch >> == dev":         true,
			"equal: [main, << pipeline.git.branch >>]": true,
			"not: << pipeline.parameters.run >>":       true,
			"false":                                    false,
		} {
			got, err := EvalWorkflowWhen(value(t, src), env, At{})
			assert.Check(t, err, "%s", src)
			assert.Check(t, cmp.Equal(got, want), "%s", src)
		}
	})
}
