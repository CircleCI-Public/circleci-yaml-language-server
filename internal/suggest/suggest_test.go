package suggest

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestClosest(t *testing.T) {
	candidates := []string{"parameters.version", "parameters.verbose", "pipeline.number"}
	for _, tc := range []struct {
		name, want string
	}{
		{name: "parameters.versoin", want: "parameters.version"},
		{name: "PIPELINE.NUMBRE", want: "pipeline.number"},
		{name: "parameters.ver", want: ""},
		{name: "parameters.version", want: ""},
	} {
		assert.Check(t, cmp.Equal(Closest(tc.name, candidates), tc.want), "closest to %q", tc.name)
	}
}
