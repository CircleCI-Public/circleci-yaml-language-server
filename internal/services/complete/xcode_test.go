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

func TestCompleteXcode(t *testing.T) {
	fake := fakes.NewCircleCI(t)
	fake.SetMachineOfferings(fakes.MachineOfferings{
		Linux: map[string][]string{"medium": {"ubuntu-2404:current"}},
		MacOS: map[string][]string{
			"m4pro.medium": {"xcode:16.4", "xcode:current", "xcode:26.0.1", "xcode:16.4.0", "xcode:26.10.0"},
		},
		Deprecated: map[string][]string{"macos": {"xcode:26.1.1"}},
	})
	settings := testHelpers.SettingsForHost(fake.URL())
	newestFirst := []string{"26.10.0", "26.0.1", "16.4.0", "16.4", "current"}

	// Each config has the line being written on line 5, and the cursor goes
	// at the end of what is written, before any closing quote.
	inJob := func(line string) string {
		return "version: 2.1\n\njobs:\n  build:\n    macos:\n" + line + "\n    resource_class: m4pro.medium\n    steps:\n      - checkout\n"
	}
	inExecutor := func(line string) string {
		return "version: 2.1\n\nexecutors:\n  mac:\n    macos:\n" + line + "\n    resource_class: m4pro.medium\n\njobs:\n  build:\n    executor: mac\n    steps:\n      - checkout\n"
	}
	cursor := func(line string) protocol.Position {
		return protocol.Position{Line: 5, Character: uint32(len(strings.TrimRight(line, `"'`)))}
	}

	for _, section := range []struct {
		name   string
		config func(string) string
	}{
		{
			name:   "a job's macos",
			config: inJob,
		},
		{
			name:   "an executor's macos",
			config: inExecutor,
		},
	} {
		t.Run(section.name, func(t *testing.T) {
			tests := []struct {
				name string
				line string
				want []string
			}{
				{
					name: "is offered the xcode key",
					line: "      xc",
					want: []string{"xcode"},
				},
				{
					name: "is offered the listed versions, newest first, at an empty version",
					line: "      xcode: ",
					want: newestFirst,
				},
				{
					name: "is offered them at a version being written",
					line: "      xcode: 16.",
					want: newestFirst,
				},
				{
					name: "is offered them at a double-quoted version",
					line: `      xcode: "16"`,
					want: newestFirst,
				},
				{
					name: "is offered them at a single-quoted version",
					line: `      xcode: ''`,
					want: newestFirst,
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					got := completionLabelsWith(t, settings, cache.New(), section.config(tt.line), cursor(tt.line))
					assert.Check(t, cmp.DeepEqual(got, tt.want))
				})
			}

			t.Run("a version replaces what is written of it, inside the quotes", func(t *testing.T) {
				line := `      xcode: "16.4"`
				pos := cursor(line)
				items := completionItemsWith(t, settings, cache.New(), section.config(line), pos)
				item := findItem(t, items, "16.4.0")
				assert.Check(t, cmp.DeepEqual(item.TextEdit, &protocol.TextEdit{
					Range: protocol.Range{
						Start: protocol.Position{Line: 5, Character: uint32(len(`      xcode: "`))},
						End:   pos,
					},
					NewText: "16.4.0",
				}))
			})
		})
	}
}
