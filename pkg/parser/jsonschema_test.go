package parser

import (
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/expect"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
	"gopkg.in/yaml.v3"
)

func Test_HandleYAMLErrors_MappingKeyError(t *testing.T) {
	content := []byte(`
anchorA: &anchorA
  A: 1

anchorB: &anchorB
  B: 2

anchorC: &anchorC
  C: 3

testFinal:
  <<: *anchorA
  <<: *anchorB
  <<: *anchorC
`)

	m := make(map[interface{}]interface{})

	err := yaml.Unmarshal(content, m)

	yamlDocument, _ := ParseContent(content)

	actualDiagnostics, err := handleYAMLErrors(err.Error(), content, yamlDocument.RootNode)

	assert.Nil(t, err)

	expectedDiagnostics := []protocol.Diagnostic{
		{
			Range:    utils.LineContentRange(12, content),
			Severity: protocol.DiagnosticSeverityError,
			Message:  `mapping key "<<" already defined at line 12`,
		},

		{
			Range:    utils.LineContentRange(13, content),
			Severity: protocol.DiagnosticSeverityError,
			Message:  `mapping key "<<" already defined at line 12`,
		},

		{
			Range:    utils.LineContentRange(13, content),
			Severity: protocol.DiagnosticSeverityError,
			Message:  `mapping key "<<" already defined at line 13`,
		},
	}

	expect.DiagnosticList(t, actualDiagnostics).To.IncludeAll(expectedDiagnostics)
}

func Test_HandleYamlError_UnknownAnchor(t *testing.T) {
	content := []byte(`
test:
  <<: *unknownAnchor
`)

	m := make(map[interface{}]interface{})

	err := yaml.Unmarshal(content, m)

	yamlDocument, _ := ParseContent(content)

	diagnostics, err := handleYAMLErrors(err.Error(), content, yamlDocument.RootNode)

	assert.Nil(t, err)

	expected := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 2, Character: 7},
			End:   protocol.Position{Line: 2, Character: 20},
		},
		Severity: protocol.DiagnosticSeverityError,
		Message:  "yaml: unknown anchor 'unknownAnchor' referenced",
	}

	expect.DiagnosticList(t, diagnostics).To.Include(expected)
}
