package validate

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
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
    resource_class: large

jobs:
  build:
    executor: macos-ios-executor
    steps:
      - checkout
`,
			Diagnostics: []protocol.Diagnostic{
				diagnostic.Error(protocol.Range{
					Start: protocol.Position{Line: 6, Character: 4},
					End:   protocol.Position{Line: 6, Character: 0x19},
				}, "Invalid resource class \"large\" for Xcode version \"26.5.0\""),
			},
		},
		{
			Name: "Xcode version from a parameter",
			YamlContent: `version: 2.1

executors:
  macos-ios-executor:
    parameters:
      xcode:
        type: string
        default: 26.5.0
    macos:
      xcode: << parameters.xcode >>
    resource_class: m4pro.medium

jobs:
  build:
    executor: macos-ios-executor
    steps:
      - checkout
`,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name: "Xcode version from an anchor and its alias",
			YamlContent: `version: 2.1

executors:
  first:
    macos:
      xcode: &xcode "26.5.0"
    resource_class: m4pro.medium
  second:
    macos:
      xcode: *xcode
    resource_class: m4pro.large

jobs:
  first:
    executor: first
    steps:
      - checkout
  second:
    executor: second
    steps:
      - checkout

workflows:
  workflow:
    jobs:
      - first
      - second
`,
			Diagnostics: []protocol.Diagnostic{},
		},
		{
			Name: "Executor name built from a parameter",
			YamlContent: `version: 2.1

executors:
  smoke-jammy:
    docker:
      - image: cimg/base:stable

jobs:
  build:
    parameters:
      codename:
        type: string
        default: jammy
    executor: smoke-<< parameters.codename >>
    steps:
      - checkout

workflows:
  main:
    jobs:
      - build
`,
			OnlyErrors:  true,
			Diagnostics: []protocol.Diagnostic{},
		},
	}

	CheckYamlErrors(t, testCases)
}

func yamlForMachine(resourceClass, image string) string {
	var builder strings.Builder
	fmt.Fprint(&builder, "version: 2.1\n")
	fmt.Fprint(&builder, "executors:\n")
	fmt.Fprint(&builder, "  toto:\n")
	if strings.Contains(resourceClass+image, "parameters.") {
		fmt.Fprint(&builder, "    parameters:\n")
	}
	for _, param := range []string{"resource_class", "ubuntu_version"} {
		if strings.Contains(resourceClass+image, "parameters."+param) {
			fmt.Fprintf(&builder, "      %s:\n        type: string\n        default: medium\n", param)
		}
	}
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
	fmt.Fprint(&builder, "jobs:\n")
	fmt.Fprint(&builder, "  build:\n")
	fmt.Fprint(&builder, "    executor: toto\n")
	fmt.Fprint(&builder, "    steps:\n")
	fmt.Fprint(&builder, "      - checkout\n")
	return builder.String()
}

// testMachineOfferings is a minimal offerings set: separate linux, windows, and macOS
// classes, so a linux class paired with a windows image is an invalid pair.
func testMachineOfferings() *circleci.Offerings {
	return &circleci.Offerings{
		Linux:   map[string][]string{"medium": {circleci.CurrentLinuxImage}},
		Windows: map[string][]string{"windows.medium": {"windows-server-2022-gui:current"}},
		MacOS: map[string][]string{
			"m4pro.medium": {"xcode:26.5.0"},
			"m4pro.large":  {"xcode:26.5.0"},
		},
		Deprecated: map[string][]string{
			"linux":   {"ubuntu-2004:2024.04.4"},
			"windows": {},
			"macos":   {"xcode:26.0.1"},
		},
	}
}

func TestUnknownMachineImageSeverity(t *testing.T) {
	severities := map[string]protocol.DiagnosticSeverity{
		"bad-image":              protocol.DiagnosticSeverityError,
		"ubunto-2404:current":    protocol.DiagnosticSeverityError,
		"windows-default":        protocol.DiagnosticSeverityError,
		"ubuntu-2404-kvm:canary": protocol.DiagnosticSeverityError,
		"ubuntu-2404:canary":     protocol.DiagnosticSeverityWarning,
		"ubuntu-2004:current":    protocol.DiagnosticSeverityWarning,
	}
	for image, want := range severities {
		t.Run(image, func(t *testing.T) {
			val := CreateValidateFromYAML(yamlForMachine("", image))
			val.Cache.MachineOfferingsCache.Set(testMachineOfferings())
			val.Validate()

			assert.Assert(t, cmp.Len(*val.Diagnostics, 1))
			got := (*val.Diagnostics)[0]
			assert.Check(t, cmp.Equal(diagnostic.MessageText(got), fmt.Sprintf("Unknown machine image %q", image)))
			assert.Check(t, cmp.Equal(got.Severity, want))
		})
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

	assert.Check(t, cmp.Len(*val.Diagnostics, 0))
}

func TestDeprecatedImageWarnings(t *testing.T) {
	testCases := []struct {
		name        string
		yamlContent string
		wantMessage string
	}{
		{
			name:        "deprecated machine image",
			yamlContent: yamlForMachine("", "ubuntu-2004:2024.04.4"),
			wantMessage: "Machine image \"ubuntu-2004:2024.04.4\" is deprecated",
		},
		{
			name: "deprecated xcode version",
			yamlContent: `version: 2.1
executors:
  macos-executor:
    macos:
      xcode: 26.0.1`,
			wantMessage: "Xcode version \"26.0.1\" is deprecated",
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			val := CreateValidateFromYAML(c.yamlContent)
			val.Cache.MachineOfferingsCache.Set(testMachineOfferings())
			val.Validate()

			var found *protocol.Diagnostic
			for i := range *val.Diagnostics {
				if (*val.Diagnostics)[i].Message == protocol.String(c.wantMessage) {
					found = &(*val.Diagnostics)[i]
					break
				}
			}

			if found == nil {
				t.Fatalf("expected diagnostic %q, got %+v", c.wantMessage, *val.Diagnostics)
			}
			assert.Check(t, cmp.Equal(protocol.DiagnosticSeverityWarning, found.Severity))
			assert.Check(t, cmp.Contains(found.Tags.Slice(), protocol.DiagnosticTagDeprecated))
		})
	}
}

func TestLegacyCircleciImages(t *testing.T) {
	testCases := []struct {
		name  string
		image string
		want  []ComparableDiagnostic
	}{
		{
			name:  "with a cimg successor of the same name",
			image: "circleci/redis:7",
			want: []ComparableDiagnostic{{
				Severity: protocol.DiagnosticSeverityWarning,
				Message:  "The legacy `circleci/redis` image is deprecated. Use `cimg/redis` instead.",
				Actions: []ComparableAction{{
					Title:  "Use `cimg/redis`",
					Writes: []string{"image: cimg/redis:7"},
				}},
			}},
		},
		{
			name:  "with a cimg successor of another name",
			image: "circleci/golang:1.17",
			want: []ComparableDiagnostic{{
				Severity: protocol.DiagnosticSeverityWarning,
				Message:  "The legacy `circleci/golang` image is deprecated. Use `cimg/go` instead.",
				Actions: []ComparableAction{{
					Title:  "Use `cimg/go`",
					Writes: []string{"image: cimg/go:1.17"},
				}},
			}},
		},
		{
			name:  "without a cimg successor",
			image: "circleci/mongo:4.2",
			want: []ComparableDiagnostic{{
				Severity: protocol.DiagnosticSeverityWarning,
				Message:  "The legacy `circleci/mongo` image is deprecated, and has no `cimg` successor.",
			}},
		},
		{
			name:  "a current circleci image",
			image: "circleci/circleci-cli:0.1.26646",
		},
		{
			name:  "an unknown circleci image",
			image: "circleci/command-convenience:0.1",
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			// With auth, the Docker Hub lookups skip the image, which leaves
			// the legacy image check's diagnostics alone.
			val := CreateValidateFromYAML(`version: 2.1
jobs:
  build:
    docker:
      - image: ` + c.image + `
        auth:
          username: $USER
          password: $PASSWORD
    steps:
      - checkout
workflows:
  main:
    jobs:
      - build
`)
			val.Validate()

			var got []ComparableDiagnostic
			for _, d := range *val.Diagnostics {
				got = append(got, diagnosticToComparableDiagnostic(d))
			}
			assert.Check(t, cmp.DeepEqual(got, c.want))
		})
	}
}

// The lookups skip an image with credentials, but the legacy image check
// needs none.
func TestCircleciNamespaceImages(t *testing.T) {
	const message = "The legacy `circleci/redis` image is deprecated. Use `cimg/redis` instead."

	testCases := []struct {
		name  string
		image string
	}{
		{
			name:  "without credentials",
			image: `- image: circleci/redis:7`,
		},
		{
			name: "with auth",
			image: `- image: circleci/redis:7
        auth:
          username: $USER
          password: $PASSWORD`,
		},
		{
			name: "with aws_auth",
			image: `- image: circleci/redis:7
        aws_auth:
          oidc_role_arn: arn:aws:iam::123456789012:role/pull`,
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			val := CreateValidateFromYAML(`version: 2.1
jobs:
  build:
    docker:
      ` + c.image + `
    steps:
      - checkout
workflows:
  main:
    jobs:
      - build
`)
			val.Validate()

			var messages []string
			for _, d := range *val.Diagnostics {
				messages = append(messages, diagnostic.MessageText(d))
			}
			assert.Check(t, cmp.Contains(messages, message))
		})
	}
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
    machine: true
jobs:
  build:
    executor: toto
    steps:
      - checkout
`,
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
			yamlContent: yamlForMachine("myorg/myrunner", circleci.CurrentLinuxImage),
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
			yamlContent: yamlForMachine("", circleci.CurrentLinuxImage),
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
				assert.Check(t, cmp.Len(*val.Diagnostics, 0))
				return
			}

			re := regexp.MustCompile(c.errRegex)

			for _, diag := range *val.Diagnostics {
				if re.MatchString(diagnostic.MessageText(diag)) {
					return
				}
			}

			t.Errorf("expected error diagnostic with message \"%s\"", c.errRegex)
		})
	}
}

func TestUnusedExecutors(t *testing.T) {
	unused := func(t *testing.T, yaml string) []string {
		t.Helper()
		var names []string
		for _, d := range *validateYAML(t, yaml) {
			if diagnostic.MessageText(d) == "Executor is unused" {
				names = append(names, strings.TrimSuffix(yaml[position.ToIndex(d.Range.Start, []byte(yaml)):position.ToIndex(d.Range.End, []byte(yaml))], ":"))
			}
		}
		return names
	}
	const executors = `version: 2.1
executors:
  small:
    docker:
      - image: cimg/base:current
  big:
    docker:
      - image: cimg/base:current
`

	t.Run("one used and one not", func(t *testing.T) {
		got := unused(t, executors+`jobs:
  build:
    executor: small
    steps: [checkout]
`)
		assert.Check(t, cmp.DeepEqual(got, []string{"big"}))
	})

	t.Run("used with parameters", func(t *testing.T) {
		got := unused(t, executors+`jobs:
  build:
    executor:
      name: small
    steps: [checkout]
  test:
    executor: { name: big }
    steps: [checkout]
`)
		assert.Check(t, cmp.Len(got, 0))
	})

	t.Run("chosen by a parameter", func(t *testing.T) {
		got := unused(t, executors+`jobs:
  build:
    parameters:
      size:
        type: enum
        enum: [small, big]
    executor: << parameters.size >>
    steps: [checkout]
workflows:
  main:
    jobs:
      - build:
          matrix:
            parameters:
              size: [small, big]
`)
		assert.Check(t, cmp.Len(got, 0))
	})

	t.Run("an executor parameter's default", func(t *testing.T) {
		got := unused(t, executors+`jobs:
  build:
    parameters:
      e:
        type: executor
        default: small
    executor: << parameters.e >>
    steps: [checkout]
workflows:
  main:
    jobs:
      - build:
          e: big
`)
		assert.Check(t, cmp.Len(got, 0))
	})

	t.Run("named only by a key", func(t *testing.T) {
		got := unused(t, executors+`jobs:
  small:
    docker:
      - image: cimg/base:current
    steps: [checkout]
workflows:
  main:
    jobs:
      - small
`)
		assert.Check(t, cmp.DeepEqual(got, []string{"big"}), "a job named small is not a use of the small executor")
	})

	t.Run("no jobs yet", func(t *testing.T) {
		got := unused(t, executors)
		assert.Check(t, cmp.Len(got, 0))
	})
}

func TestOverriddenExecutorSettings(t *testing.T) {
	const message = "resource_class is set both on the job and on the executor; the job's value is used " +
		"and the executor's is ignored. See https://circleci.com/docs/reference/configuration-reference/#executors"
	severities := func(t *testing.T, yaml string) []protocol.DiagnosticSeverity {
		t.Helper()
		var got []protocol.DiagnosticSeverity
		for _, d := range *validateYAML(t, yaml) {
			if diagnostic.MessageText(d) == message {
				got = append(got, d.Severity)
			}
		}
		return got
	}
	const config = `version: 2.1
executors:
  box:
    docker:
      - image: cimg/base:current
    resource_class: large
  alias-box: box
jobs:
  big:
    executor: box
    resource_class: xlarge
    steps: [checkout]
`

	t.Run("every use overrides it", func(t *testing.T) {
		got := severities(t, config+`  bigger:
    executor:
      name: box
    resource_class: 2xlarge
    steps: [checkout]
`)
		assert.Check(t, cmp.DeepEqual(got, []protocol.DiagnosticSeverity{
			protocol.DiagnosticSeverityWarning, protocol.DiagnosticSeverityWarning,
		}))
	})

	t.Run("another job uses the executor's", func(t *testing.T) {
		got := severities(t, config+`  small:
    executor: box
    steps: [checkout]
`)
		assert.Check(t, cmp.DeepEqual(got, []protocol.DiagnosticSeverity{protocol.DiagnosticSeverityHint}))
	})

	t.Run("an alias uses the executor's", func(t *testing.T) {
		got := severities(t, config+`  small:
    executor: alias-box
    steps: [checkout]
`)
		assert.Check(t, cmp.DeepEqual(got, []protocol.DiagnosticSeverity{protocol.DiagnosticSeverityHint}))
	})

	t.Run("a parameter can use the executor's", func(t *testing.T) {
		got := severities(t, config+`  any:
    parameters:
      executor:
        type: executor
        default: box
    executor: << parameters.executor >>
    steps: [checkout]
`)
		assert.Check(t, cmp.DeepEqual(got, []protocol.DiagnosticSeverity{protocol.DiagnosticSeverityHint}))
	})
}
