package validate

import (
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/dockerhub"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
)

type ComparableAction struct {
	Title  string
	Writes []string
}

type ComparableDiagnostic struct {
	Severity protocol.DiagnosticSeverity
	Message  string
	Actions  []ComparableAction
}

type DockerHubMock struct {
	NoExist  bool
	NoLatest bool
	NoTag    bool
	Tags     []string
}

func (me DockerHubMock) DoesImageExist(namespace, image string) bool {
	return !me.NoExist
}

func (me DockerHubMock) GetImageTags(namespace, image string) ([]string, error) {
	if me.Tags == nil {
		return []string{}, nil
	}
	return me.Tags, nil
}

func (me DockerHubMock) ImageHasTag(namespace, image, tag string) bool {
	if tag == "latest" {
		return !me.NoLatest
	}
	return !me.NoTag
}

func TestValidateDockerImage(t *testing.T) {
	testCases := []struct {
		Name        string
		YamlContent string
		MockAPI     dockerhub.DockerHubAPI
		Diagnostics []ComparableDiagnostic
	}{
		{
			Name: "Should give no diagnostic on a valid docker image",

			Diagnostics: []ComparableDiagnostic{},

			MockAPI: DockerHubMock{},

			YamlContent: `version: 2.1

executors:
  some-executor:
    docker:
      - image: namespace/image:tag`,
		},
		{
			Name: "Should give error on non-existing image",

			Diagnostics: []ComparableDiagnostic{
				{
					Severity: protocol.DiagnosticSeverityError,
					Message:  "Docker image not found \"namespace/image:tag\"",
				},
			},

			MockAPI: DockerHubMock{
				NoExist: true,
			},

			YamlContent: `version: 2.1

executors:
  some-executor:
    docker:
      - image: namespace/image:tag`,
		},
		{
			Name: "Should give an error on non-existing tag",

			Diagnostics: []ComparableDiagnostic{
				{
					Severity: protocol.DiagnosticSeverityError,
					Message:  "Docker image \"namespace/image:tag\" has no tag \"tag\"",
				},
			},

			MockAPI: DockerHubMock{
				NoLatest: true,
				NoTag:    true,
			},

			YamlContent: `version: 2.1

executors:
  some-executor:
    docker:
      - image: namespace/image:tag`,
		},
		{
			Name: "Should give error on existing image non tagged image with no latest",

			Diagnostics: []ComparableDiagnostic{
				{
					Severity: protocol.DiagnosticSeverityError,
					Message:  "Docker image \"namespace/image\" has no tag \"latest\"",
				},
			},

			MockAPI: DockerHubMock{NoLatest: true},

			YamlContent: `version: 2.1

executors:
  some-executor:
    docker:
      - image: namespace/image`,
		},
		{
			Name: "Should give hint on existing non tagged image with no latest with tags",

			Diagnostics: []ComparableDiagnostic{
				{
					Severity: protocol.DiagnosticSeverityError,
					Message:  "Docker image \"namespace/image\" has no tag \"latest\"",
					Actions: []ComparableAction{
						{
							Title:  "Use last tag",
							Writes: []string{":v1.2.3"},
						},
					},
				},
			},

			MockAPI: DockerHubMock{
				NoLatest: true,
				Tags:     []string{"tagname", "v1.2.3"},
			},

			YamlContent: `version: 2.1

executors:
  some-executor:
    docker:
      - image: namespace/image`,
		},
		{
			Name: "Should work on jobs",

			Diagnostics: []ComparableDiagnostic{},

			MockAPI: DockerHubMock{},

			YamlContent: `version: 2.1

jobs:
  some-job:
    docker:
      - image: namespace/image:tag

workflows:
  someworkflow:
    jobs:
      - some-job
`,
		},
		{
			Name: "Should work on jobs with parameter",

			Diagnostics: []ComparableDiagnostic{},

			MockAPI: DockerHubMock{},

			YamlContent: `version: 2.1

jobs:
  some-job:
    parameters:
      image-tag:
        type: string
    docker:
      - image: namespace/image:<< parameters.image-tag >>

workflows:
  someworkflow:
    jobs:
      - some-job:
          image-tag: tag
`,
		},
		{
			Name: "Should give no diagnostic on valid SHA256 digest with tag",

			Diagnostics: []ComparableDiagnostic{},

			MockAPI: DockerHubMock{},

			YamlContent: `version: 2.1

executors:
  some-executor:
    docker:
      - image: cimg/node:22.11.0@sha256:76aae59c6259672ab68819b8960de5ef571394681089eab2b576f85f080c73ba`,
		},
		{
			Name: "Should give no diagnostic on valid SHA256 digest without tag",

			Diagnostics: []ComparableDiagnostic{},

			MockAPI: DockerHubMock{},

			YamlContent: `version: 2.1

executors:
  some-executor:
    docker:
      - image: cimg/node@sha256:76aae59c6259672ab68819b8960de5ef571394681089eab2b576f85f080c73ba`,
		},
		{
			Name: "Should give error on invalid digest format - too short",

			Diagnostics: []ComparableDiagnostic{
				{
					Severity: protocol.DiagnosticSeverityError,
					Message:  "Invalid Docker image digest format \"foo\". Expected format: sha256:<64 hex characters>",
				},
			},

			MockAPI: DockerHubMock{},

			YamlContent: `version: 2.1

executors:
  some-executor:
    docker:
      - image: cimg/go:1.24@foo`,
		},
		{
			Name: "Should give error on invalid digest format - wrong prefix",

			Diagnostics: []ComparableDiagnostic{
				{
					Severity: protocol.DiagnosticSeverityError,
					Message:  "Invalid Docker image digest format \"abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890\". Expected format: sha256:<64 hex characters>",
				},
			},

			MockAPI: DockerHubMock{},

			YamlContent: `version: 2.1

executors:
  some-executor:
    docker:
      - image: cimg/node:18@abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890`,
		},
		{
			Name: "Should give error on invalid digest format - wrong hash length",

			Diagnostics: []ComparableDiagnostic{
				{
					Severity: protocol.DiagnosticSeverityError,
					Message:  "Invalid Docker image digest format \"sha256:abc123\". Expected format: sha256:<64 hex characters>",
				},
			},

			MockAPI: DockerHubMock{},

			YamlContent: `version: 2.1

executors:
  some-executor:
    docker:
      - image: cimg/go:latest@sha256:abc123`,
		},
		{
			Name: "Should give error on invalid digest format - non-hex characters",

			Diagnostics: []ComparableDiagnostic{
				{
					Severity: protocol.DiagnosticSeverityError,
					Message:  "Invalid Docker image digest format \"sha256:ghijklmnopqrstuvwxyz1234567890abcdef1234567890abcdef1234567890\". Expected format: sha256:<64 hex characters>",
				},
			},

			MockAPI: DockerHubMock{},

			YamlContent: `version: 2.1

executors:
  some-executor:
    docker:
      - image: node:alpine@sha256:ghijklmnopqrstuvwxyz1234567890abcdef1234567890abcdef1234567890`,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			val := CreateValidateFromYAML(tt.YamlContent)
			val.APIs = ValidateAPIs{
				DockerHub: tt.MockAPI,
			}

			val.Validate()

			diags := *val.Diagnostics
			compareDiagnostics(t, tt.Diagnostics, diags)
		})
	}
}

func TestChooseTagToRecommend(t *testing.T) {
	testCases := []struct {
		Name   string
		Tags   []string
		Output string
	}{
		{
			Name:   "Empty list should return empty string",
			Tags:   []string{},
			Output: "",
		},
		{
			Name:   "Latest tags should be filtered",
			Tags:   []string{"latest"},
			Output: "",
		},
		{
			Name:   "Should give the first semver",
			Tags:   []string{"tagname", "v1.2.3"},
			Output: "v1.2.3",
		},
		{
			Name:   "Should give the first version if no semver",
			Tags:   []string{"tagname", "other-tagname"},
			Output: "tagname",
		},
		{
			Name:   "Should handle versions without the 'v'",
			Tags:   []string{"tagname", "1.2.3"},
			Output: "1.2.3",
		},
		{
			Name:   "Should handle lighter versions",
			Tags:   []string{"tagname", "1.2"},
			Output: "1.2",
		},
		{
			Name:   "Should handle even lighter versions",
			Tags:   []string{"tagname", "1"},
			Output: "1",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			assert.Equal(t, tt.Output, chooseTagToRecommend(tt.Tags))
		})
	}
}

func TestCreateTagTextEdit(t *testing.T) {
	tag := "16.20.1-browsers"
	img := ast.DockerImage{
		Image: ast.DockerImageInfo{
			Namespace: "cimg",
			Name:      "node",
			Tag:       "14",
			FullPath:  "cimg/node:14",
		},
		ImageRange: protocol.Range{
			Start: protocol.Position{
				Character: 9,
				Line:      48,
			},
			End: protocol.Position{
				Character: 28,
				Line:      48,
			},
		},
		Name:        "",
		Entrypoint:  []string{},
		Command:     []string{},
		User:        "",
		Environment: map[string]string{},
		Auth:        ast.DockerImageAuth{},
		AwsAuth:     ast.DockerImageAWSAuth{},
	}
	testCases := []struct {
		Name   string
		Img    ast.DockerImage
		Tag    string
		Output protocol.TextEdit
	}{
		{
			Name: "Image tag action should replace the correct string",
			Img:  img,
			Tag:  tag,
			Output: protocol.TextEdit{
				NewText: ":" + tag,
				Range: protocol.Range{
					Start: protocol.Position{
						Character: 25,
						Line:      48,
					},
					End: protocol.Position{
						Character: 26 + uint32(len(tag)),
						Line:      48,
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			assert.Equal(t, tt.Output, createTagTextEdit(&img, tag))
		})
	}
}

func compareDiagnostics(t *testing.T, expected []ComparableDiagnostic, diagnostics []protocol.Diagnostic) {
	actual := []ComparableDiagnostic{}

	for _, diag := range diagnostics {
		actual = append(actual, diagnosticToComparableDiagnostic(diag))
	}

	assert.Equal(t, expected, actual)
}

func diagnosticToComparableDiagnostic(diag protocol.Diagnostic) ComparableDiagnostic {
	actions, ok := diag.Data.([]protocol.CodeAction)
	var codeActions []ComparableAction

	if ok && len(actions) > 0 {
		codeActions = make([]ComparableAction, 0)

		for _, action := range actions {
			edits := []string{}

			for _, edit := range action.Edit.Changes {
				for _, change := range edit {
					edits = append(edits, change.NewText)
				}
			}
			codeActions = append(codeActions, ComparableAction{
				Title:  action.Title,
				Writes: edits,
			})
		}
	}
	return ComparableDiagnostic{
		Severity: diag.Severity,
		Message:  diag.Message,
		Actions:  codeActions,
	}
}
