package stepoutputs

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestContains(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{name: "a reference", text: "${{ steps.build.outputs.digest }}", want: true},
		{name: "a reference with no spaces", text: "${{steps.build.outputs.digest}}", want: true},
		{name: "a reference to a list item", text: "${{ steps.build.outputs.tags[0] }}", want: true},
		{name: "ids and names with dashes and underscores", text: "${{ steps.build-image.outputs.image_ref }}", want: true},
		{name: "a reference in text", text: "registry/app@${{ steps.build.outputs.digest }}:latest", want: true},
		{name: "the singular form is not a reference", text: "${{ step.build.output.digest }}", want: false},
		{name: "steps without outputs is not a reference", text: "${{ steps.build.digest }}", want: false},
		{name: "another expression is not a reference", text: "${{ github.sha }}", want: false},
		{name: "a parameter reference is not a reference", text: "<< parameters.digest >>", want: false},
		{name: "an id that starts with a digit is not a reference", text: "${{ steps.1build.outputs.digest }}", want: false},
		{name: "an unclosed reference is not a reference", text: "${{ steps.build.outputs.digest", want: false},
		{name: "a negative index is not a reference", text: "${{ steps.build.outputs.tags[-1] }}", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Check(t, cmp.Equal(Contains(test.text), test.want))
		})
	}
}
