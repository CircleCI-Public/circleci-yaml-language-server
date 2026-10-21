package compile

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestRenderPartial(t *testing.T) {
	env := Env{"matrix.os": "linux", "matrix.go": "1.26", "matrix.debug": true}
	for _, tc := range []struct {
		text, want string
	}{
		{text: "test-<< matrix.os >>-<< matrix.go >>", want: "test-linux-1.26"},
		{text: "test-<<matrix.os>>", want: "test-linux"},
		{text: "test-<< matrix.os >>-<< pipeline.number >>", want: "test-linux-<< pipeline.number >>"},
		{text: "<< matrix.os >>-<< parameters.x or matrix.go >>", want: "linux-<< parameters.x or matrix.go >>"},
		{text: `\<< matrix.os >>`, want: `\<< matrix.os >>`},
		{text: `dir\\<< matrix.os >>`, want: `dir\linux`},
		{text: "a<<# matrix.debug >>-debug<</ matrix.debug >>", want: "a-debug"},
		{text: "a<<# other.flag >>-<< matrix.os >><</ other.flag >>", want: "a<<# other.flag >>-<< matrix.os >><</ other.flag >>"},
		{text: "unclosed << matrix.os", want: "unclosed << matrix.os"},
		{text: "plain", want: "plain"},
	} {
		assert.Check(t, cmp.Equal(RenderPartial(tc.text, env), tc.want), "%q", tc.text)
	}
}
