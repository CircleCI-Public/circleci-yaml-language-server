package parser_test

import (
	"strings"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/expect"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/testHelpers"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestJobExecutorMachineTrueOnApp(t *testing.T) {
	yaml := `version: 2.1
jobs:
  test:
    machine: true
    steps:
      - checkout
`

	yamlDocument, err := parser.ParseFromContent(
		[]byte(yaml),
		testHelpers.GetDefaultLsContext(),
		uri.File(""),
		protocol.Position{},
	)

	assert.Equal(t, err, nil)
	assert.True(t, yamlDocument.Context.Api.UseDefaultInstance())
	img := utils.GetLatestUbuntu2204Image()
	machineRange := protocol.Range{
		Start: protocol.Position{Line: 3, Character: 4},
		End:   protocol.Position{Line: 3, Character: 17},
	}
	expect.DiagnosticList(t, *yamlDocument.Diagnostics).To.Include(protocol.Diagnostic{
		Range:    machineRange,
		Severity: protocol.DiagnosticSeverityWarning,
		Message:  utils.GetMachineTrueMessage(img),
		Data: []protocol.CodeAction{
			utils.CreateCodeActionTextEdit("Replace with most updated ubuntu image", yamlDocument.URI,
				[]protocol.TextEdit{
					{
						Range: machineRange,
						NewText: `machine:
		` + strings.Repeat(" ", int(machineRange.Start.Character)) + `  image: ` + utils.GetLatestUbuntu2204Image(),
					},
				}, false),
		},
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

	yamlDocument, err := parser.ParseFromContent(
		[]byte(yaml),
		testHelpers.GetDefaultLsContext(),
		uri.File(""),
		protocol.Position{},
	)

	assert.Equal(t, err, nil)
	assert.True(t, yamlDocument.Context.Api.UseDefaultInstance())
	assert.Empty(t, *yamlDocument.Diagnostics)
}

func TestJobExecutorMachineTrueOnSelfHosted(t *testing.T) {
	yaml := `version: 2.1
jobs:
  test:
    machine: true
    steps:
      - checkout
`

	yamlDocument, err := parser.ParseFromContent(
		[]byte(yaml),
		testHelpers.GetLsContextForHost("https://mycircleci.example.com"),
		uri.File(""),
		protocol.Position{},
	)

	assert.Equal(t, err, nil)
	assert.False(t, yamlDocument.Context.Api.UseDefaultInstance())
	assert.Empty(t, *yamlDocument.Diagnostics)
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

	yamlDocument, err := parser.ParseFromContent(
		[]byte(yaml),
		testHelpers.GetDefaultLsContext(),
		uri.File(""),
		protocol.Position{},
	)

	assert.Equal(t, err, nil)
	assert.True(t, yamlDocument.Context.Api.UseDefaultInstance())
	img := utils.GetLatestUbuntu2204Image()
	machineRange := protocol.Range{
		Start: protocol.Position{Line: 7, Character: 4},
		End:   protocol.Position{Line: 7, Character: 17},
	}
	expect.DiagnosticList(t, *yamlDocument.Diagnostics).To.Include(
		protocol.Diagnostic{
			Range:    machineRange,
			Severity: protocol.DiagnosticSeverityWarning,
			Message:  utils.GetMachineTrueMessage(img),
			Data: []protocol.CodeAction{
				utils.CreateCodeActionTextEdit("Replace with most updated ubuntu image", yamlDocument.URI,
					[]protocol.TextEdit{
						{
							Range: machineRange,
							NewText: `machine:
		` + strings.Repeat(" ", int(machineRange.Start.Character)) + `  image: ` + utils.GetLatestUbuntu2204Image(),
						},
					}, false),
			},
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

	yamlDocument, err := parser.ParseFromContent(
		[]byte(yaml),
		testHelpers.GetDefaultLsContext(),
		uri.File(""),
		protocol.Position{},
	)

	assert.Equal(t, err, nil)
	assert.True(t, yamlDocument.Context.Api.UseDefaultInstance())
	assert.Empty(t, *yamlDocument.Diagnostics)
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

	yamlDocument, err := parser.ParseFromContent(
		[]byte(yaml),
		testHelpers.GetDefaultLsContext(),
		uri.File(""),
		protocol.Position{},
	)

	assert.Equal(t, err, nil)
	assert.True(t, yamlDocument.Context.Api.UseDefaultInstance())
	assert.Empty(t, *yamlDocument.Diagnostics)
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

	yamlDocument, err := parser.ParseFromContent(
		[]byte(yaml),
		testHelpers.GetDefaultLsContext(),
		uri.File(""),
		protocol.Position{},
	)

	assert.Equal(t, err, nil)
	assert.True(t, yamlDocument.Context.Api.UseDefaultInstance())
	img := utils.GetLatestUbuntu2204Image()
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
			Message:  utils.GetMachineTrueMessage(img),
			Data: []protocol.CodeAction{
				utils.CreateCodeActionTextEdit("Replace with most updated ubuntu image", yamlDocument.URI,
					[]protocol.TextEdit{
						{
							Range: machineRange,
							NewText: `machine:
		` + strings.Repeat(" ", int(machineRange.Start.Character)) + `  image: ` + utils.GetLatestUbuntu2204Image(),
						},
					}, false),
			},
		},
	)
}

func TestIsFromUnfetchableOrb(t *testing.T) {
	yamlDocument, err := parser.ParseFromContent([]byte(`version: 2.1

orbs:
  slack: circleci/slack@4.12.5
  ccc: cci-dev/ccc@<<pipeline.parameters.dev-orb-version>>
`), testHelpers.GetDefaultLsContext(), uri.File(""), protocol.Position{})

	assert.Nil(t, err)
	assert.True(t, yamlDocument.IsFromUnfetchableOrb("ccc/entity"))
	assert.False(t, yamlDocument.IsFromUnfetchableOrb("slack/entity"))
}
