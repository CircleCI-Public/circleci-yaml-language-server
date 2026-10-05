package circleci

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestResourceClassSize(t *testing.T) {
	cases := []struct {
		class ResourceClass
		want  string
	}{
		{
			class: ResourceClass{CPU: 2, RAMMB: 8192},
			want:  "2 vCPUs, 8 GB RAM",
		},
		{
			class: ResourceClass{CPU: 1, RAMMB: 2048},
			want:  "1 vCPU, 2 GB RAM",
		},
		{
			class: ResourceClass{CPU: 4, RAMMB: 15360},
			want:  "4 vCPUs, 15 GB RAM",
		},
		// GPU Gen2 Linux Small is a few megabytes short of 16 GB.
		{
			class: ResourceClass{CPU: 4, RAMMB: 16380},
			want:  "4 vCPUs, 16 GB RAM",
		},
		{
			class: ResourceClass{CPU: 3, RAMMB: 6656},
			want:  "3 vCPUs, 6.5 GB RAM",
		},
	}
	for _, tc := range cases {
		got := tc.class.Size()
		assert.Check(t, cmp.Equal(got, tc.want), "%+v", tc.class)
	}
}
