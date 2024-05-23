package validate

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
)

func TestExecutorValidation(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name: "Ignore workflow's jobs that are come from uncheckable orbs",
			YamlContent: `version: 2.1

parameters:
  dev-orb-version:
    type: string
    default: "dev:alpha"

orbs:
  ccc: cci-dev/ccc@<<pipeline.parameters.dev-orb-version>>

jobs:
  job:
    executor: ccc/executor
    steps:
      - run: echo "Hello"

workflows:
  someworkflow:
    jobs:
      - job
`,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name: "flag resource class error",
			YamlContent: `version: 2.1

executors:
  macos-ios-executor:
    macos:
      xcode: "15.1.0"
    resource_class: large`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 6, Character: 4},
					End:   protocol.Position{Line: 6, Character: 0x19},
				}, "Invalid resource class \"large\" for Xcode version \"15.1.0\""),
			},
		},
	}

	CheckYamlErrors(t, testCases)
}

func yamlForMachine(resourceClass, image string) string {
	var builder strings.Builder
	fmt.Fprint(&builder, "version: 2.1\n")
	fmt.Fprint(&builder, "executors:\n")
	fmt.Fprint(&builder, "  toto:\n")
	if resourceClass == "" && image == "" {
		fmt.Fprint(&builder, "    machine: {}\n")
	} else {
		fmt.Fprint(&builder, "    machine:\n")
		if resourceClass != "" {
			fmt.Fprintf(&builder, "      resource_class: %#v\n", resourceClass)
		}
		if image != "" {
			fmt.Fprintf(&builder, "      image: %#v\n", image)
		}
	}
	return builder.String()
}

func TestMachineExecutor(t *testing.T) {
	type testCase struct {
		name        string
		yamlContent string
		errRegex    string
	}
	testCases := []testCase{
		{
			name: "machine:true",
			yamlContent: `version: 2.1
		executors:
		  toto:
		    machine: true`,
		},
		{
			name:        "rc:undefined img:undefined",
			yamlContent: yamlForMachine("", ""),
		},
		{
			name:        "rc:self-hosted img:undefined",
			yamlContent: yamlForMachine("myorg/myrunner", ""),
		},
		{
			name:        "rc:param img:undefined",
			yamlContent: yamlForMachine("<< parameters.resource_class >>", ""),
		},
		{
			name:        "rc:linux img:undefined",
			yamlContent: yamlForMachine(utils.ValidLinuxResourceClasses[0], ""),
		},
		{
			name:        "rc:windows img:undefined",
			yamlContent: yamlForMachine(utils.ValidWindowsResourceClasses[0], ""),
		},
		{
			name:        "rc:toto img:undefined",
			yamlContent: yamlForMachine("toto", ""),
			errRegex:    "Unknown resource class",
		},
		{
			name:        "rc:self-hosted img:current",
			yamlContent: yamlForMachine("myorg/myrunner", utils.CurrentLinuxImage),
			errRegex:    "Extraneous image",
		},
		{
			name:        "rc:param img:param",
			yamlContent: yamlForMachine("<< parameters.resource_class >>", "ubuntu:<< parameters.ubuntu_version >>"),
		},
		{
			name:        "rc:undefined img:param",
			yamlContent: yamlForMachine("", "ubuntu:<< parameters.ubuntu_version >>"),
		},
		{
			name:        "rc:undefined img:linux",
			yamlContent: yamlForMachine("", utils.CurrentLinuxImage),
		},
		{
			name:        "rc:windows img:windows",
			yamlContent: yamlForMachine(utils.ValidWindowsResourceClasses[0], utils.ValidWindowsImages[0]),
		},
		{
			name:        "rc:undefined img:toto",
			yamlContent: yamlForMachine("", "toto"),
			errRegex:    "Unknown machine image",
		},
		{
			name:        "rc:linux img:toto",
			yamlContent: yamlForMachine(utils.ValidLinuxResourceClasses[0], "toto"),
			errRegex:    "Unknown machine image",
		},
		{
			name:        "rc:toto img:toto",
			yamlContent: yamlForMachine("toto", "toto"),
			errRegex:    "Unknown machine image",
		},
		{
			name:        "rc:toto img:toto",
			yamlContent: yamlForMachine("toto", "toto"),
			errRegex:    "Unknown resource class",
		},
		{
			name:        "rc:linux img:windows",
			yamlContent: yamlForMachine(utils.ValidLinuxResourceClasses[0], utils.ValidWindowsImages[0]),
			errRegex:    "Machine image \".*?\" is not available for",
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			val := CreateValidateFromYAML(c.yamlContent)
			val.Validate()

			if c.errRegex == "" {
				assert.Len(t, *val.Diagnostics, 0)
				return
			}

			re := regexp.MustCompile(c.errRegex)

			for _, diag := range *val.Diagnostics {
				if re.MatchString(diag.Message) {
					return
				}
			}

			t.Errorf("expected error diagnostic with message \"%s\"", c.errRegex)
		})
	}
}
