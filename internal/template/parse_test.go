package template

import (
	"testing"

	"github.com/CircleCI-Public/expr/parsers"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestParse(t *testing.T) {
	ignoreExpr := cmpopts.IgnoreFields(Tag{}, "Expr")

	t.Run("text, tags and sections, with where they are", func(t *testing.T) {
		const text = "echo << a >> && <<# b.c >>x<<^ d >>y<</ d >><</ b.c >> done >> log"
		got, problem := Parse(text)
		assert.Assert(t, cmp.Nil(problem))
		assert.Check(t, cmp.DeepEqual(got, Template{
			Text{Value: "echo "},
			Tag{Name: "a", Start: 5, End: 12},
			Text{Value: " && "},
			Section{Name: "b.c", Start: 16, End: 54, Body: Template{
				Text{Value: "x"},
				Section{Name: "d", Inverted: true, Start: 27, End: 44, Body: Template{Text{Value: "y"}}},
			}},
			Text{Value: " done >> log"},
		}, ignoreExpr))
	})

	t.Run("a tag keeps its expression", func(t *testing.T) {
		got, problem := Parse(`<< pipeline.git.branch == "main" >>`)
		assert.Assert(t, cmp.Nil(problem))
		assert.Assert(t, cmp.Len(got, 1))
		tag, ok := got[0].(Tag)
		assert.Assert(t, ok)
		_, isBinary := tag.Expr.(parsers.Binary)
		assert.Check(t, isBinary, "expression is %T", tag.Expr)
		assert.Check(t, cmp.Equal(tag.Name, ""), "an expression has no name")
	})

	t.Run("a bare name has its expression and its name", func(t *testing.T) {
		got, problem := Parse("<< pipeline.number >>")
		assert.Assert(t, cmp.Nil(problem))
		assert.Assert(t, cmp.Len(got, 1))
		tag := got[0].(Tag)
		assert.Check(t, tag.Expr != nil)
		assert.Check(t, cmp.Equal(tag.Name, "pipeline.number"))
	})

	t.Run("a name expressions don't allow has no expression", func(t *testing.T) {
		got, problem := Parse("<< pipeline.parameters.url:with:colons >>")
		assert.Assert(t, cmp.Nil(problem))
		assert.Assert(t, cmp.Len(got, 1))
		tag := got[0].(Tag)
		assert.Check(t, cmp.Nil(tag.Expr))
		assert.Check(t, cmp.Equal(tag.Name, "pipeline.parameters.url:with:colons"))
	})

	t.Run("escapes stay as written", func(t *testing.T) {
		got, problem := Parse(`cat \<<EOF \\<< a >>`)
		assert.Assert(t, cmp.Nil(problem))
		assert.Check(t, cmp.DeepEqual(got, Template{
			Text{Value: `cat \<<EOF \`},
			Tag{Name: "a", Start: 13, End: 20},
		}, ignoreExpr))
	})

	t.Run("a problem gives no template", func(t *testing.T) {
		got, problem := Parse("ok << a >> then <<# b >>unclosed")
		assert.Assert(t, problem != nil)
		assert.Check(t, cmp.Contains(problem.Message, "Unclosed section"))
		assert.Check(t, cmp.Nil(got))
	})

	t.Run("no text is an empty template", func(t *testing.T) {
		got, problem := Parse("")
		assert.Assert(t, cmp.Nil(problem))
		assert.Check(t, cmp.Len(got, 0))
	})
}
