package validate

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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
      xcode: 26.5.0
    resource_class: large`,
			Diagnostics: []protocol.Diagnostic{
				utils.CreateErrorDiagnosticFromRange(protocol.Range{
					Start: protocol.Position{Line: 6, Character: 4},
					End:   protocol.Position{Line: 6, Character: 0x19},
				}, "Invalid resource class \"large\" for Xcode version \"26.5.0\""),
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

// testMachineOfferings is a minimal offerings set: separate linux, windows, and macOS
// classes, so a linux class paired with a windows image is an invalid pair.
func testMachineOfferings() *utils.Offerings {
	return &utils.Offerings{
		Linux:   map[string][]string{"medium": {utils.CurrentLinuxImage}},
		Windows: map[string][]string{"windows.medium": {"windows-server-2022-gui:current"}},
		MacOS: map[string][]string{
			"m4pro.medium": {"26.5.0"},
			"m4pro.large":  {"26.5.0"},
		},
	}
}

// When the offerings API is unavailable, machine validation is skipped rather than
// flagging valid images as errors.
func TestMachineExecutorSkipsWhenOfferingsUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	val := CreateValidateFromYAML(yamlForMachine("toto", "bogus:image"))
	val.Context.Api.HostUrl = server.URL
	val.Validate()

	assert.Len(t, *val.Diagnostics, 0)
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
			yamlContent: yamlForMachine("medium", ""),
		},
		{
			name:        "rc:windows img:undefined",
			yamlContent: yamlForMachine("windows.medium", ""),
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
			yamlContent: yamlForMachine("windows.medium", "windows-server-2022-gui:current"),
		},
		{
			name:        "rc:undefined img:toto",
			yamlContent: yamlForMachine("", "toto"),
			errRegex:    "Unknown machine image",
		},
		{
			name:        "rc:linux img:toto",
			yamlContent: yamlForMachine("medium", "toto"),
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
			yamlContent: yamlForMachine("medium", "windows-server-2022-gui:current"),
			errRegex:    "Machine image \".*?\" is not available for",
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			val := CreateValidateFromYAML(c.yamlContent)
			val.Cache.MachineOfferingsCache.Set(testMachineOfferings())
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
