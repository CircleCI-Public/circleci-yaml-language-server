package template

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestCheckAcceptsTemplates(t *testing.T) {
	for _, text := range []string{
		"no tags at all",
		"echo << parameters.name >>",
		"<<parameters.name>>",
		"<< pipeline.git.branch == \"main\" and pipeline.number > 10 >>",
		"<< not (pipeline.parameters.deploy or pipeline.git.tag) >>",
		"<< pipeline.git.branch starts-with \"release/\" >>",
		"<< pipeline.parameters.url:with:colons >>",
		`cat \<<EOF`,
		"echo done >> log.txt",
		"<<# parameters.debug >>set -x<</ parameters.debug >>",
		"<<^ parameters.debug >>quiet<</ parameters.debug >>",
		"<< matrix.os >>-<< matrix.version >>",
	} {
		problem, found := Check(text)
		assert.Check(t, !found, "%q: %v", text, problem)
	}
}

func TestCheckFindsProblems(t *testing.T) {
	for _, tc := range []struct {
		text    string
		at      string
		message string
	}{
		{text: "cat <<EOF > file", at: "<<", message: "Unclosed '<<' tag"},
		{text: "echo << parameters.name", at: "<<", message: "Unclosed '<<' tag"},
		{text: "echo <<>>", at: "<<", message: "Expected expression after '<<'"},
		{text: "echo << >>", at: " >>", message: "Expected expression"},
		{text: "<< pipeline.git.branch == >>", at: " pipeline.git.branch == ", message: "Expected expression"},
		{text: "<< a and and b >>", at: "and b", message: `Expected expression, found "and"`},
		{text: "<< (a or b >>", at: " ", message: "Expected ')' after expression"},
		{text: "<< a & b >>", at: "&", message: "Unexpected character '&'"},
		{text: `<< a == "open >>`, at: `"`, message: "Unterminated string"},
		{text: "<<# parameters.debug >>set -x", at: "<<#", message: "Unclosed section, expected '<</'"},
		{text: "<<# a >>x<</ a >><</ b >>", at: "<</ b", message: "Expected section end tag to match start tag"},
		{text: "<<# not valid >>x<</ not valid >>", at: " not valid ", message: "Expected valid identifier"},
		{text: "<< " + strings.Repeat("a", 2049) + " >>", at: " a", message: "Expressions must be less than 2048 characters"},
	} {
		problem, found := Check(tc.text)
		assert.Assert(t, found, "%q", tc.text)
		assert.Check(t, cmp.Contains(problem.Message, tc.message), "%q", tc.text)
		assert.Check(t, cmp.Equal(problem.Start, strings.Index(tc.text, tc.at)), "%q starts at %q", tc.text, tc.at)
	}
}

func TestReferences(t *testing.T) {
	t.Run("the names in expressions and sections, with where they are", func(t *testing.T) {
		const text = `<< a >>, << not (pipeline.git.branch == "main" or b.c) >>, <<# d.e >>x<</ d.e >>`
		got := References(text)

		names := []string{}
		for _, reference := range got {
			names = append(names, reference.Name)
			assert.Check(t, cmp.Equal(text[reference.Start:reference.End], reference.Name))
		}
		assert.Check(t, cmp.DeepEqual(names, []string{"a", "pipeline.git.branch", "b.c", "d.e", "d.e"}))
	})

	t.Run("a tag that isn't an expression is one name", func(t *testing.T) {
		got := References("<<  pipeline.parameters.a:b >>")
		assert.Check(t, cmp.DeepEqual(got, []Reference{{Name: "pipeline.parameters.a:b", Start: 4, End: 27}}))
	})

	t.Run("up to the first problem", func(t *testing.T) {
		got := References("<< a >> << and >> << b >>")
		assert.Check(t, cmp.DeepEqual(got, []Reference{{Name: "a", Start: 3, End: 4}}))
	})
}

func TestIsOnlyTag(t *testing.T) {
	for text, want := range map[string]bool{
		"<< pipeline.number >>":                           true,
		`<< pipeline.git.branch == "main" and 10 or 1 >>`: true,
		"<<parameters.count>>":                            true,
		"<< a >> << b >>":                                 false,
		"x-<< a >>":                                       false,
		"<< a and and b >>":                               false,
		"<<# a >>x<</ a >>":                               false,
		"plain":                                           false,
	} {
		got := IsOnlyTag(text)
		assert.Check(t, cmp.Equal(got, want), "%q", text)
	}
}
