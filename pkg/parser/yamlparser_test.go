package parser_test

import (
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/expect"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/testHelpers"
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

	expect.DiagnosticList(t, *yamlDocument.Diagnostics).To.Include(protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 3, Character: 4},
			End:   protocol.Position{Line: 3, Character: 17},
		},
		Severity: protocol.DiagnosticSeverityWarning,
		Message:  "Using `machine: true` is deprecated, please instead specify an image to use.",
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
	expect.DiagnosticList(t, *yamlDocument.Diagnostics).To.Include(
		protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: 7, Character: 4},
				End:   protocol.Position{Line: 7, Character: 17},
			},
			Severity: protocol.DiagnosticSeverityWarning,
			Message:  "Using `machine: true` is deprecated, please instead specify an image to use.",
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
	expect.DiagnosticList(
		t,
		*yamlDocument.Diagnostics,
	).To.Include(
		protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: 3, Character: 4},
				End:   protocol.Position{Line: 3, Character: 17},
			},
			Severity: protocol.DiagnosticSeverityWarning,
			Message:  "Using `machine: true` is deprecated, please instead specify an image to use.",
		},
	)
}
