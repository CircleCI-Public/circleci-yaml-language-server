package parser_test

import (
	"slices"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/codeaction"
	parser2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/expect"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/tsalloc"
)

func TestErrCacheMissing(t *testing.T) {
	c := cache.New()
	_, err := parser2.ParseFromUriWithCache(uri.MustParse("file:///toto.yaml"), c, nil)

	assert.Check(t, cmp.ErrorIs(err, parser2.ErrCacheMissing))
}

func TestJobExecutorMachineTrueOnApp(t *testing.T) {
	yaml := `version: 2.1
jobs:
  test:
    machine: true
    steps:
      - checkout
`

	yamlDocument, err := parser2.ParseFromContent(
		[]byte(yaml),
		testHelpers.DefaultSettings(),
		uri.File(""),
		protocol.Position{},
	)

	assert.Check(t, err)
	assert.Check(t, yamlDocument.Context.Api.UseDefaultInstance())
	img := circleci.CurrentLinuxImage
	machineRange := protocol.Range{
		Start: protocol.Position{Line: 3, Character: 4},
		End:   protocol.Position{Line: 3, Character: 17},
	}
	expect.DiagnosticList(t, *yamlDocument.Diagnostics).To.Include(protocol.Diagnostic{
		Range:    machineRange,
		Severity: protocol.DiagnosticSeverityWarning,
		Message:  protocol.String(parser2.MachineTrueMessage(img)),
		Data: codeaction.Data([]protocol.CodeAction{
			codeaction.TextEdit("Replace with current Ubuntu image", yamlDocument.URI,
				[]protocol.TextEdit{
					{
						Range:   machineRange,
						NewText: "machine:\n" + strings.Repeat(" ", int(machineRange.Start.Character)) + "  image: " + circleci.CurrentLinuxImage,
					},
				}, false),
		}),
	})
}

func TestJobExecutorMachineFalseOnApp(t *testing.T) {
	yaml := `version: 2.1
jobs:
  test:
    machine: false
    steps:
      - checkout
`

	yamlDocument, err := parser2.ParseFromContent(
		[]byte(yaml),
		testHelpers.DefaultSettings(),
		uri.File(""),
		protocol.Position{},
	)

	assert.Check(t, err)
	assert.Check(t, yamlDocument.Context.Api.UseDefaultInstance())
	assert.Check(t, cmp.Len(*yamlDocument.Diagnostics, 0))
}

func TestJobExecutorMachineTrueOnSelfHosted(t *testing.T) {
	yaml := `version: 2.1
jobs:
  test:
    machine: true
    steps:
      - checkout
`

	yamlDocument, err := parser2.ParseFromContent(
		[]byte(yaml),
		testHelpers.SettingsForHost("https://mycircleci.example.com"),
		uri.File(""),
		protocol.Position{},
	)

	assert.Check(t, err)
	assert.Check(t, !yamlDocument.Context.Api.UseDefaultInstance())
	assert.Check(t, cmp.Len(*yamlDocument.Diagnostics, 0))
}

func TestJobExecutorMachineTrueOnPublicRunner(t *testing.T) {
	yaml := `version: 2.1
executors:
  linux-13:
    docker:
      - image: cimg/node:13.13
jobs:
  test:
    machine: true
    resource_class: large
    steps:
      - checkout
`

	yamlDocument, err := parser2.ParseFromContent(
		[]byte(yaml),
		testHelpers.DefaultSettings(),
		uri.File(""),
		protocol.Position{},
	)

	assert.Check(t, err)
	assert.Check(t, yamlDocument.Context.Api.UseDefaultInstance())
	img := circleci.CurrentLinuxImage
	machineRange := protocol.Range{
		Start: protocol.Position{Line: 7, Character: 4},
		End:   protocol.Position{Line: 7, Character: 17},
	}
	expect.DiagnosticList(t, *yamlDocument.Diagnostics).To.Include(
		protocol.Diagnostic{
			Range:    machineRange,
			Severity: protocol.DiagnosticSeverityWarning,
			Message:  protocol.String(parser2.MachineTrueMessage(img)),
			Data: codeaction.Data([]protocol.CodeAction{
				codeaction.TextEdit("Replace with current Ubuntu image", yamlDocument.URI,
					[]protocol.TextEdit{
						{
							Range:   machineRange,
							NewText: "machine:\n" + strings.Repeat(" ", int(machineRange.Start.Character)) + "  image: " + circleci.CurrentLinuxImage,
						},
					}, false),
			}),
		},
	)
}

func TestJobExecutorMachineTrueOnPrivateRunner(t *testing.T) {
	yaml := `version: 2.1
jobs:
  test:
    machine: true
    resource_class: private/runner
    steps:
      - checkout
`

	yamlDocument, err := parser2.ParseFromContent(
		[]byte(yaml),
		testHelpers.DefaultSettings(),
		uri.File(""),
		protocol.Position{},
	)

	assert.Check(t, err)
	assert.Check(t, yamlDocument.Context.Api.UseDefaultInstance())
	assert.Check(t, cmp.Len(*yamlDocument.Diagnostics, 0))
}

func TestMachineTrueOnWindows(t *testing.T) {
	machineRange := protocol.Range{
		Start: protocol.Position{Line: 3, Character: 4},
		End:   protocol.Position{Line: 3, Character: 17},
	}
	configs := map[string]string{
		"in a job": `version: 2.1
jobs:
  test:
    machine: true
    resource_class: windows.medium
    steps:
      - checkout
`,
		"in an executor": `version: 2.1
executors:
  windows:
    machine: true
    resource_class: windows.medium

jobs:
  test:
    executor: windows
    steps:
      - checkout
`,
	}

	for name, yaml := range configs {
		t.Run(name, func(t *testing.T) {
			yamlDocument, err := parser2.ParseFromContent(
				[]byte(yaml),
				testHelpers.DefaultSettings(),
				uri.File(""),
				protocol.Position{},
			)
			assert.NilError(t, err)

			var machineTrue []protocol.Diagnostic
			for _, diagnostic := range *yamlDocument.Diagnostics {
				if diagnostic.Range == machineRange {
					machineTrue = append(machineTrue, diagnostic)
				}
			}
			assert.Assert(t, cmp.Len(machineTrue, 1))

			diagnostic := machineTrue[0]
			assert.Check(t, cmp.Equal(diagnostic.Severity, protocol.DiagnosticSeverityWarning))
			assert.Check(t, cmp.DeepEqual(diagnostic.Message, protocol.String(parser2.MachineTrueWindowsMessage)))
			assert.Check(t, cmp.DeepEqual(diagnostic.Data, codeaction.Data(nil)), "no quick fix, since the right image isn't known")
		})
	}
}

func TestExecutorWithDefinedMachine(t *testing.T) {
	yaml := `version: 2.1

executors:
  machine-test:
    machine:
      image: node:alpine

jobs:
  test:
    executor: machine-test
    steps:
      - checkout
`

	yamlDocument, err := parser2.ParseFromContent(
		[]byte(yaml),
		testHelpers.DefaultSettings(),
		uri.File(""),
		protocol.Position{},
	)

	assert.Check(t, err)
	assert.Check(t, yamlDocument.Context.Api.UseDefaultInstance())
	assert.Check(t, cmp.Len(*yamlDocument.Diagnostics, 0))
}

func TestExecutorWithMachineTrue(t *testing.T) {
	yaml := `version: 2.1
executors:
  machine-test:
    machine: true

jobs:
  test:
    executor: machine-test
    steps:
      - checkout
`

	yamlDocument, err := parser2.ParseFromContent(
		[]byte(yaml),
		testHelpers.DefaultSettings(),
		uri.File(""),
		protocol.Position{},
	)

	assert.Check(t, err)
	assert.Check(t, yamlDocument.Context.Api.UseDefaultInstance())
	img := circleci.CurrentLinuxImage
	machineRange := protocol.Range{
		Start: protocol.Position{Line: 3, Character: 4},
		End:   protocol.Position{Line: 3, Character: 17},
	}
	expect.DiagnosticList(
		t,
		*yamlDocument.Diagnostics,
	).To.Include(
		protocol.Diagnostic{
			Range:    machineRange,
			Severity: protocol.DiagnosticSeverityWarning,
			Message:  protocol.String(parser2.MachineTrueMessage(img)),
			Data: codeaction.Data([]protocol.CodeAction{
				codeaction.TextEdit("Replace with current Ubuntu image", yamlDocument.URI,
					[]protocol.TextEdit{
						{
							Range:   machineRange,
							NewText: "machine:\n" + strings.Repeat(" ", int(machineRange.Start.Character)) + "  image: " + circleci.CurrentLinuxImage,
						},
					}, false),
			}),
		},
	)
}

func TestIsFromUnfetchableOrb(t *testing.T) {
	yamlDocument, err := parser2.ParseFromContent([]byte(`version: 2.1

orbs:
  slack: circleci/slack@4.12.5
  ccc: cci-dev/ccc@<<pipeline.parameters.dev-orb-version>>
  chosen: << pipeline.parameters.orb >>
`), testHelpers.DefaultSettings(), uri.File(""), protocol.Position{})

	assert.Check(t, err)
	assert.Check(t, yamlDocument.IsFromUnfetchableOrb(t.Context(), "ccc/entity", cache.New()))
	assert.Check(t, yamlDocument.IsFromUnfetchableOrb(t.Context(), "chosen/entity", cache.New()))
	assert.Check(t, !yamlDocument.IsFromUnfetchableOrb(t.Context(), "slack/entity", cache.New()))
}

func TestSetupKey(t *testing.T) {
	type TestCase struct {
		Content     string
		ExpectValue bool
		ExpectRange protocol.Range
		Name        string
	}
	// These tests represent the behaviour of the CCI product.
	// You can see the different thing that have been tried on here:
	// https://app.circleci.com/pipelines/github/circleci/devex-demo?branch=continuation-workflows
	testCases := []TestCase{
		{
			Name: "Is true when set to true",
			Content: `version: 2.1

setup: true

jobs:
  toto:
    docker:
      - image: cimg/go:1.19.1
    steps:
      - run: echo "Hello world"`,
			ExpectValue: true,
			ExpectRange: protocol.Range{
				Start: protocol.Position{
					Line:      2,
					Character: 0,
				},
				End: protocol.Position{
					Line:      2,
					Character: 11,
				},
			},
		},
		{
			Name: "Is false when not set",
			Content: `version: 2.1

jobs:
  toto:
    docker:
      - image: cimg/go:1.19.1
    steps:
      - run: echo "Hello world"`,
			ExpectValue: false,
		},
		{
			Name: "Is true with complex values",
			Content: `version: 2.1

setup:
  complex:
    values: 42

jobs:
  toto:
    docker:
      - image: cimg/go:1.19.1
    steps:
      - run: echo "Hello world"`,
			ExpectValue: true,
			ExpectRange: protocol.Range{
				Start: protocol.Position{
					Line:      2,
					Character: 0,
				},
				End: protocol.Position{
					Line:      4,
					Character: 14,
				},
			},
		},
		{
			Name: "Is false when empty",
			Content: `version: 2.1

setup:

jobs:
  toto:
    docker:
      - image: cimg/go:1.19.1
    steps:
      - run: echo "Hello world"`,
			ExpectValue: false,
			ExpectRange: protocol.Range{
				Start: protocol.Position{Line: 0x2, Character: 0x0},
				End:   protocol.Position{Line: 0x2, Character: 0x6},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			yamlDocument, err := parser2.ParseFromContent([]byte(tt.Content), testHelpers.DefaultSettings(), uri.File(""), protocol.Position{})
			assert.Check(t, err)
			assert.Check(t, cmp.Equal(tt.ExpectValue, yamlDocument.Setup))
			assert.Check(t, cmp.DeepEqual(tt.ExpectRange, yamlDocument.SetupRange))
		})
	}
}

func TestModifyTextForAutocomplete(t *testing.T) {
	// Each case used to panic: the one on the root node because it has no
	// parent, and the other because the node there has no text.
	for name, tc := range map[string]struct {
		content string
		pos     protocol.Position
	}{
		"on the root node between documents": {
			content: "version: 2.1\n---\njobs:\n  build:\n---\n",
			pos:     protocol.Position{Line: 0, Character: 13},
		},
		"at the start of an empty block scalar": {
			content: "version: 2.1\njobs:\n  build:\n    steps:\n      - run: |\n",
			pos:     protocol.Position{Line: 4, Character: 0},
		},
	} {
		t.Run(name, func(t *testing.T) {
			doc, err := parser2.ParseFromContent([]byte(tc.content), testHelpers.DefaultSettings(), uri.File(""), protocol.Position{})
			assert.NilError(t, err)
			t.Cleanup(doc.Close)

			modified := slices.Collect(doc.ModifyTextForAutocomplete(tc.pos))

			assert.Assert(t, len(modified) != 0)
			last := modified[len(modified)-1]
			assert.Check(t, cmp.Equal(last.Tag, "original"))
			for _, m := range modified[:len(modified)-1] {
				m.Document.Close()
			}
		})
	}
}

func TestModifyTextForAutocompleteYieldsCopiesAsTheyAreAskedFor(t *testing.T) {
	const content = "version: 2.1\njobs:\n  build:\n    steps:\n      - checkout\n"
	// At the end of "checkout", where all three copies parse cleanly.
	pos := protocol.Position{Line: 4, Character: 16}

	parse := func(t *testing.T) parser2.YamlDocument {
		t.Helper()
		doc, err := parser2.ParseFromContent([]byte(content), testHelpers.DefaultSettings(), uri.File(""), protocol.Position{})
		assert.NilError(t, err)
		t.Cleanup(doc.Close)
		return doc
	}

	t.Run("each copy, then the original", func(t *testing.T) {
		doc := parse(t)

		var tags []string
		for modified := range doc.ModifyTextForAutocomplete(pos) {
			tags = append(tags, modified.Tag)
			if modified.Tag != "original" {
				modified.Document.Close()
			}
		}

		assert.Check(t, cmp.DeepEqual(tags, []string{"edit-item", "edit-key", "edit-value", "original"}))
	})

	t.Run("stopping at the first copy leaves no tree open", func(t *testing.T) {
		doc := parse(t)
		leaked := tsalloc.Track(t)

		for modified := range doc.ModifyTextForAutocomplete(pos) {
			modified.Document.Close()
			break
		}

		assert.Check(t, cmp.Equal(leaked(), int64(0)), "tree-sitter allocations left open")
	})

	// Completion points the document it iterates from at each copy in turn,
	// and each copy must still be the original with one placeholder in it.
	t.Run("each copy is made from the document as it was", func(t *testing.T) {
		current := parse(t)
		at := strings.Index(content, "checkout") + len("checkout")

		var copies []parser2.YamlDocument
		t.Cleanup(func() {
			for _, doc := range copies {
				doc.Close()
			}
		})
		for modified := range current.ModifyTextForAutocomplete(pos) {
			if modified.Tag == "original" {
				assert.Check(t, cmp.Equal(string(modified.Document.Content), content))
				break
			}
			want := content[:at] + modified.Diff + content[at:]
			assert.Check(t, cmp.Equal(string(modified.Document.Content), want), modified.Tag)
			copies = append(copies, modified.Document)
			current = modified.Document
		}

		assert.Check(t, cmp.Len(copies, 3))
	})
}

func TestInsertText(t *testing.T) {
	const content = "a: é\nb: 2\n"

	for name, tc := range map[string]struct {
		pos  protocol.Position
		want string
	}{
		"before the character at the position": {
			pos:  protocol.Position{Line: 1, Character: 3},
			want: "a: é\nb: X2\n",
		},
		"after a character of more than one byte": {
			pos:  protocol.Position{Line: 0, Character: 5},
			want: "a: éX\nb: 2\n",
		},
		"nowhere at the end of the content": {
			pos:  protocol.Position{Line: 2, Character: 0},
			want: content,
		},
	} {
		t.Run(name, func(t *testing.T) {
			doc, err := parser2.ParseFromContent([]byte(content), testHelpers.DefaultSettings(), uri.File(""), protocol.Position{})
			assert.NilError(t, err)
			t.Cleanup(doc.Close)

			edited, err := doc.InsertText(tc.pos, "X")
			assert.NilError(t, err)
			t.Cleanup(edited.Close)

			assert.Check(t, cmp.Equal(string(edited.Content), tc.want))
		})
	}
}
