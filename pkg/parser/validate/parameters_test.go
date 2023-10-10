package validate

import (
	"os"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func TestJobParameterType(t *testing.T) {
	correctParamFilePath := "./testdata/correct_param_type.yml"
	correctParamFileContent, err := os.ReadFile(correctParamFilePath)
	if err != nil {
		panic(err)
	}
	wrongParamFilePath := "./testdata/wrong_param_type.yml"
	wrongParamFileContent, err2 := os.ReadFile(wrongParamFilePath)
	if err2 != nil {
		panic(err2)
	}
	wrongParamIntegerFilePath := "./testdata/wrong_param_type_integer.yml"
	wrongParamIntegerFileContent, err2 := os.ReadFile(wrongParamIntegerFilePath)
	if err2 != nil {
		panic(err2)
	}
	wrongParamBooleanFilePath := "./testdata/wrong_param_type_boolean.yml"
	wrongParamBooleanFileContent, err2 := os.ReadFile(wrongParamBooleanFilePath)
	if err2 != nil {
		panic(err2)
	}
	testCases := []ValidateTestCase{
		{
			Name:        "Using a global Parameter on a job parameter with the same type definition should not result in error",
			YamlContent: string(correctParamFileContent),
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name:        "Parameter usage should error when param usage is different from param definition",
			YamlContent: string(wrongParamFileContent),
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 24, Character: 9},
					End:   protocol.Position{Line: 24, Character: 54},
				}, "Parameter skip for build must be a string"),
			},
		},
		{
			Name:        "Parameter usage should error when param usage is different from param definition",
			YamlContent: string(wrongParamIntegerFileContent),
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 24, Character: 9},
					End:   protocol.Position{Line: 24, Character: 54},
				}, "Parameter skip for build must be a boolean"),
			},
		},
		{
			Name:        "Parameter usage should error when param usage is different from param definition",
			YamlContent: string(wrongParamBooleanFileContent),
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 24, Character: 9},
					End:   protocol.Position{Line: 24, Character: 54},
				}, "Parameter skip for build must be a boolean"),
			},
		},
	}

	CheckYamlErrors(t, testCases)
}
