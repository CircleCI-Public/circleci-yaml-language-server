package utils

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

// The probe costs a request, so it is skipped for the one host known to serve
// the V3 orb routes. Every other host, CircleCI Server included, is asked.
func TestV3OrbRoutesAssumed(t *testing.T) {
	for _, testCase := range []struct {
		host string
		want bool
	}{
		{CIRCLE_CI_APP_HOST_URL, true},
		{"https://circleci.example.com", false},
		{"https://circleci.com.example.com", false},
		{"http://localhost:8080", false},
		{"", false},
	} {
		got := v3OrbRoutesAssumed(testCase.host)
		assert.Check(t, cmp.Equal(got, testCase.want), "host %q", testCase.host)
	}
}
