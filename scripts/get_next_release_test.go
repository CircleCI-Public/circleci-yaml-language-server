package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIncrementVersion(t *testing.T) {
	type TestCase struct {
		name     string
		input    string
		expected string
	}
	testCases := []TestCase{
		{
			name:     "From normal tag",
			input:    "1.2.3",
			expected: "1.2.4-pre.1",
		},
		{
			name:     "From normal tag with metadata",
			input:    "1.2.3+meta",
			expected: "1.2.4-pre.1",
		},
		{
			name:     "From prerelease tag",
			input:    "1.2.3-pre.1",
			expected: "1.2.3-pre.2",
		},
		{
			name:     "From prerelease tag with metadata",
			input:    "1.2.3-pre.1+meta",
			expected: "1.2.3-pre.2+meta",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			value := incrementVersion(tt.input)
			assert.Equal(t, tt.expected, value)
		})
	}
}
