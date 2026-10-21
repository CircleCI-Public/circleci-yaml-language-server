package ir

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestLocOf(t *testing.T) {
	tests := []struct {
		name    string
		content string
		line    int
		byteCol int
		want    Loc
	}{
		{
			name:    "ascii",
			content: "version: 2.1\njobs:\n  build: {}\n",
			line:    2, byteCol: 2,
			want: Loc{Source: "c.yml", Line: 3, Col: 3},
		},
		{
			name:    "multi-byte prefix",
			content: "a: 1\nname: \"héllo ✓\" # x\n",
			line:    1, byteCol: len(`name: "héllo ✓" `),
			want: Loc{Source: "c.yml", Line: 2, Col: 17},
		},
		{
			name:    "multi-byte on earlier line only",
			content: "# ✓✓✓\nkey: v\n",
			line:    1, byteCol: 5,
			want: Loc{Source: "c.yml", Line: 2, Col: 6},
		},
		{
			name: "inline orb in parent coordinates",
			content: "orbs:\n" +
				"  my-orb:\n" +
				"    commands:\n" +
				"      héllo:\n" +
				"        steps: [checkout]\n",
			line: 3, byteCol: len("      hé"),
			want: Loc{Source: "c.yml", Line: 4, Col: 9},
		},
		{
			name:    "first line",
			content: "jobs: {}",
			want:    Loc{Source: "c.yml", Line: 1, Col: 1},
		},
		{
			name:    "column past end of line",
			content: "ab\ncd\n",
			line:    0, byteCol: 4,
			want: Loc{Source: "c.yml", Line: 1, Col: 5},
		},
		{
			name:    "line past end of content",
			content: "ab\n",
			line:    5, byteCol: 2,
			want: Loc{Source: "c.yml", Line: 6, Col: 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LocOf("c.yml", []byte(tt.content), tt.line, tt.byteCol)
			assert.Check(t, cmp.DeepEqual(got, tt.want))
		})
	}
}
