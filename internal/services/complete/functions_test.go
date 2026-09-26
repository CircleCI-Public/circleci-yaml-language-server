package complete

import (
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/fakes"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func TestCompleteFunctionVersion(t *testing.T) {
	const config = `version: 2.1

functions:
  setup-go: github.com/circleci-functions/setup-go@v0
  unknown: github.com/circleci-functions/unknown@

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
      - setup-go
`
	fake := fakes.NewCircleCI(t)
	fake.AddFunction("fn-setup-go", "github.com/circleci-functions/setup-go", "Install a Go toolchain.",
		fakes.FunctionVersion{ID: "ver-1", Version: "v0.5.1-684fd5b", Descriptor: map[string]any{"name": "setup-go"}},
		fakes.FunctionVersion{ID: "ver-2", Version: "v0.6.0-1a2b3c4", Descriptor: map[string]any{"name": "setup-go"}},
	)
	settings := testHelpers.SettingsForHost(fake.URL())

	t.Run("a function's version is offered its published versions", func(t *testing.T) {
		pos := protocol.Position{Line: 3, Character: uint32(len("  setup-go: github.com/circleci-functions/setup-go@v0"))}
		got := completionLabelsWith(t, settings, cache.New(), config, pos)
		assert.Check(t, cmp.DeepEqual(got, []string{
			"v0.5.1-684fd5b", "v0.6.0-1a2b3c4",
		}))
	})

	t.Run("an unpublished function's is offered none", func(t *testing.T) {
		pos := protocol.Position{Line: 4, Character: uint32(len("  unknown: github.com/circleci-functions/unknown@"))}
		got := completionLabelsWith(t, settings, cache.New(), config, pos)
		assert.Check(t, cmp.Len(got, 0))
	})
}

func TestCompleteFunctionFlags(t *testing.T) {
	fake := fakes.NewCircleCI(t)
	fake.AddFunction("fn-setup-go", "github.com/circleci-functions/setup-go", "Install a Go toolchain.",
		fakes.FunctionVersion{ID: "ver-1", Version: "v0.5.1-684fd5b", Descriptor: map[string]any{
			"name": "setup-go",
			"flags": []map[string]any{
				{"name": "version", "type": "string", "description": "The Go version."},
				{"name": "cache", "type": "bool"},
			},
			"commands": map[string]any{
				"lint": map[string]any{"flags": []map[string]any{{"name": "fix", "type": "bool"}}},
			},
		}},
	)
	settings := testHelpers.SettingsForHost(fake.URL())

	// The cursor goes at the end of the step's last line.
	withStep := func(step string) (string, protocol.Position) {
		config := `version: 2.1

functions:
  setup-go: github.com/circleci-functions/setup-go@v0.5.1-684fd5b

jobs:
  build:
    docker:
      - image: cimg/base:stable
    steps:
` + step
		lines := strings.Split(config, "\n")
		last := len(lines) - 1
		return config, protocol.Position{Line: uint32(last), Character: uint32(len(lines[last]))}
	}

	tests := []struct {
		name string
		step string
		want []string
	}{
		{"a function's flags", "      - setup-go:\n          with:\n            ", []string{"version", "cache"}},
		{"the flags it doesn't pass yet", "      - setup-go:\n          with:\n            version: \"1.27\"\n            ", []string{"cache"}},
		{"a command's flags", "      - setup-go/lint:\n          with:\n            ", []string{"fix"}},
		{"a boolean flag's values", "      - setup-go:\n          with:\n            cache: ", []string{"true", "false"}},
		{"no values for other flags", "      - setup-go:\n          with:\n            version: ", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, pos := withStep(tt.step)
			got := completionLabelsWith(t, settings, cache.New(), config, pos)
			assert.Check(t, cmp.DeepEqual(got, tt.want))
		})
	}
}
