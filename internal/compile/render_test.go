package compile

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/ir"
)

func TestRender(t *testing.T) {
	env := Env{
		"parameters.name":    "world",
		"parameters.count":   int64(3),
		"parameters.debug":   true,
		"parameters.off":     false,
		"parameters.nothing": nil,
		"parameters.list":    []any{"a", "b"},
		"parameters.map":     map[string]any{"k": "v"},
		"pipeline.git.tag":   "",
	}

	t.Run("a single tag gives its value, typed", func(t *testing.T) {
		for _, tc := range []struct {
			text string
			want any
		}{
			{text: "<< parameters.count >>", want: int64(3)},
			{text: "<< parameters.debug >>", want: true},
			{text: "<< parameters.nothing >>", want: nil},
			{text: "<< parameters.list >>", want: []any{"a", "b"}},
			{text: "<< parameters.map >>", want: map[string]any{"k": "v"}},
			{text: "<< parameters.debug and 1 or 0 >>", want: int64(1)},
			{text: `<< parameters.off or "fallback" >>`, want: "fallback"},
		} {
			got, err := Render(tc.text, env, At{})
			assert.Check(t, err, "%q", tc.text)
			assert.Check(t, cmp.DeepEqual(got, tc.want), "%q", tc.text)
		}
	})

	t.Run("a single tag with an unknown variable gives nil", func(t *testing.T) {
		got, err := Render("<< parameters.missing >>", env, At{})
		assert.Check(t, err)
		assert.Check(t, cmp.Nil(got))
	})

	t.Run("a tag with no variable is text", func(t *testing.T) {
		got, err := Render("<< 1 >>", env, At{})
		assert.Check(t, err)
		assert.Check(t, cmp.Equal(got, "1"))
	})

	t.Run("text without a tag is as written", func(t *testing.T) {
		got, err := Render("echo hi\r\n", env, At{})
		assert.Check(t, err)
		assert.Check(t, cmp.Equal(got, "echo hi\r\n"))
	})

	t.Run("tags in text", func(t *testing.T) {
		for _, tc := range []struct {
			text, want string
		}{
			{text: "hello << parameters.name >>", want: "hello world"},
			{text: "n=<< parameters.count >> d=<< parameters.debug >> x=<< parameters.nothing >>", want: "n=3 d=true x="},
			{text: "[<< parameters.missing >>]", want: "[]"},
			{text: `cd << parameters.nothing or "hello" >>`, want: "cd hello"},
			{text: `cd << parameters.missing or "hello" >>`, want: "cd "},
			{text: `cat \<<EOF << parameters.name >>`, want: `cat \<<EOF world`},
			{text: `dir\\<< parameters.name >>`, want: `dir\world`},
		} {
			got, err := RenderString(tc.text, env, At{})
			assert.Check(t, err, "%q", tc.text)
			assert.Check(t, cmp.Equal(got, tc.want), "%q", tc.text)
		}
	})

	t.Run("sections", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			v    any
			want bool
		}{
			{name: "nil", v: nil, want: false},
			{name: "false", v: false, want: false},
			{name: "empty", v: "", want: false},
			{name: "empty list", v: []any{}, want: false},
			{name: "space", v: "   ", want: true},
			{name: "zero", v: int64(0), want: true},
			{name: "empty map", v: map[string]any{}, want: true},
			{name: "true", v: true, want: true},
		} {
			e := Env{"parameters.x": tc.v}
			got, err := RenderString("[<<# parameters.x >>yes<</ parameters.x >>]", e, At{})
			assert.Check(t, err)
			assert.Check(t, cmp.Equal(got == "[yes]", tc.want), "section on %s", tc.name)
			got, err = RenderString("[<<^ parameters.x >>yes<</ parameters.x >>]", e, At{})
			assert.Check(t, err)
			assert.Check(t, cmp.Equal(got == "[yes]", !tc.want), "inverted section on %s", tc.name)
		}
		got, err := RenderString("[<<# parameters.missing >>yes<</ parameters.missing >>]", env, At{})
		assert.Check(t, err)
		assert.Check(t, cmp.Equal(got, "[]"))
	})

	t.Run("a line holding only a section tag adds no line", func(t *testing.T) {
		const text = "a\r\n  <<# parameters.debug >>\nb\n<</ parameters.debug >>\nc\n"
		got, err := RenderString(text, env, At{})
		assert.Check(t, err)
		assert.Check(t, cmp.Equal(got, "a\nb\nc\n"))
	})

	t.Run("errors", func(t *testing.T) {
		at := At{Loc: ir.Loc{Source: "config.yml", Line: 4, Col: 9}}
		for _, tc := range []struct {
			text, message string
		}{
			{text: "x << parameters.list >>", message: "a list can't be written into text"},
			{text: "<< parameters.list == 1 >>", message: "parameters.list is a list, which can't be used in an expression"},
			{text: "<< parameters.name > 1 >>", message: `Expected numeric operands to ">" operator`},
			{text: "<< parameters.name", message: "Unclosed '<<' tag"},
		} {
			_, err := Render(tc.text, env, at)
			assert.Check(t, cmp.ErrorContains(err, tc.message), "%q", tc.text)
			assert.Check(t, cmp.ErrorContains(err, "at config.yml:4:9"), "%q", tc.text)
		}
	})
}

func TestUnknown(t *testing.T) {
	env := Env{"parameters.version": "1", "pipeline.number": int64(1)}

	t.Run("none", func(t *testing.T) {
		assert.Check(t, Unknown("<< parameters.version >>", env, At{}))
	})

	t.Run("every unknown name, once, with what it's probably a typo of", func(t *testing.T) {
		err := Unknown("<< parameters.versoin >> << parameters.other >> <<# parameters.versoin >>x<</ parameters.versoin >>", env, At{})
		assert.Check(t, cmp.Error(err, "Unknown variable(s): parameters.versoin (did you mean parameters.version?), parameters.other"))
	})
}

func TestUnescape(t *testing.T) {
	for text, want := range map[string]string{
		`cat \<<EOF`:   `cat <<EOF`,
		`a \>> b`:      `a >> b`,
		`keep \\<< it`: `keep \\<< it`,
		`no escapes`:   `no escapes`,
		`\`:            `\`,
	} {
		assert.Check(t, cmp.Equal(Unescape(text), want), "%q", text)
	}
}

func TestFormatDouble(t *testing.T) {
	for f, want := range map[float64]string{
		1.5:     "1.5",
		3:       "3.0",
		0:       "0.0",
		1e20:    "1.0E20",
		1.25e-5: "1.25E-5",
		-2e7:    "-2.0E7",
	} {
		assert.Check(t, cmp.Equal(formatDouble(f), want), "%v", f)
	}
}
