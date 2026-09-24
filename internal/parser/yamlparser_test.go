package parser_test

import (
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
			codeaction.TextEdit("Replace with most updated ubuntu image", yamlDocument.URI,
				[]protocol.TextEdit{
					{
						Range: machineRange,
						NewText: `machine:
		` + strings.Repeat(" ", int(machineRange.Start.Character)) + `  image: ` + circleci.CurrentLinuxImage,
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
				codeaction.TextEdit("Replace with most updated ubuntu image", yamlDocument.URI,
					[]protocol.TextEdit{
						{
							Range: machineRange,
							NewText: `machine:
		` + strings.Repeat(" ", int(machineRange.Start.Character)) + `  image: ` + circleci.CurrentLinuxImage,
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
				codeaction.TextEdit("Replace with most updated ubuntu image", yamlDocument.URI,
					[]protocol.TextEdit{
						{
							Range: machineRange,
							NewText: `machine:
		` + strings.Repeat(" ", int(machineRange.Start.Character)) + `  image: ` + circleci.CurrentLinuxImage,
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
`), testHelpers.DefaultSettings(), uri.File(""), protocol.Position{})

	assert.Check(t, err)
	assert.Check(t, yamlDocument.IsFromUnfetchableOrb("ccc/entity"))
	assert.Check(t, !yamlDocument.IsFromUnfetchableOrb("slack/entity"))
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
