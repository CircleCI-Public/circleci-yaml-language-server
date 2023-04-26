package validate

import (
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func TestValidateDockerImage(t *testing.T) {
	testCases := []ValidateTestCase{
		{
			Name:        "Docker images should be allowed not to have image tag",
			OnlyErrors:  false,
			Diagnostics: []protocol.Diagnostic{},
			YamlContent: `version: 2.1,

jobs:
  somejob:
    docker:
      - image: cimg/node
    steps:
      - run: echo "Hello world"

workflows:
  someworkflow:
    jobs:
      - somejob
`,
		},

		{
			Name:       "Non existing docker images should show error",
			OnlyErrors: false,
			Diagnostics: []protocol.Diagnostic{utils.CreateErrorDiagnosticFromRange(protocol.Range{
				Start: protocol.Position{Line: 5, Character: 8},
				End:   protocol.Position{Line: 5, Character: 38},
			}, "Docker image not found cimg/non-existing-image")},
			YamlContent: `version: 2.1,

jobs:
  somejob:
    docker:
      - image: cimg/non-existing-image
    steps:
      - run: echo "Hello world"

workflows:
  someworkflow:
    jobs:
      - somejob
`,
		},
	}

	CheckYamlErrors(t, testCases)
}
