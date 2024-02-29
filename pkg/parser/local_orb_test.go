package parser_test

import (
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/dockerhub"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser/validate"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/testHelpers"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestLocalOrbJob(t *testing.T) {
	content := `version: 2.1

orbs:
  localorb:
    jobs:
      localjob:
        parameters:
            name:
                default: "world"
                type: string
        docker:
          - image: cimg/base:2020.01
        steps:
          - run: echo "Hello << parameter.name >>"`
	doc := GetDocForTests(t, content, "localorb")
	jobKey := "localjob"
	assert.Contains(t, doc.Jobs, jobKey)
	job := doc.Jobs[jobKey]
	assert.EqualValues(t, job.Range.Start.Line, 5)

	// Test parameter
	parameterKey := "name"
	assert.Contains(t, job.Parameters, parameterKey)
	parameter := job.Parameters[parameterKey]
	assert.EqualValues(t, parameter.GetRange().Start.Line, 7)

	// Test docker
	assert.EqualValues(t, job.Docker.Name, "docker")
	assert.Len(t, job.Docker.Image, 1)
	image := job.Docker.Image[0]
	assert.EqualValues(t, image.ImageRange.Start.Line, 11)

	// Test step
	assert.Len(t, job.Steps, 1)
	step := job.Steps[0]
	assert.NotNil(t, step)
	assert.EqualValues(t, step.GetRange().Start.Line, 13)
}

func TestLocalOrbJobWithComment(t *testing.T) {
	content := `version: 2.1

orbs:
  localorb:
#    commands:
#      localcommand:
#        steps:
#          - run: echo "Hello world"

    jobs:
      localjob:
        parameters:
            name:
                default: "world"
                type: string
        docker:
          - image: cimg/base:2020.01
        steps:
          - run: echo "Hello << parameter.name >>"`
	doc := GetDocForTests(t, content, "localorb")
	jobKey := "localjob"
	assert.Contains(t, doc.Jobs, jobKey)
	job := doc.Jobs[jobKey]
	assert.EqualValues(t, job.Range.Start.Line, 10)

	// Test parameter
	parameterKey := "name"
	assert.Contains(t, job.Parameters, parameterKey)
	parameter := job.Parameters[parameterKey]
	assert.EqualValues(t, parameter.GetRange().Start.Line, 12)

	// Test docker
	assert.EqualValues(t, job.Docker.Name, "docker")
	assert.Len(t, job.Docker.Image, 1)
	image := job.Docker.Image[0]
	assert.EqualValues(t, image.ImageRange.Start.Line, 16)

	// Test step
	assert.Len(t, job.Steps, 1)
	step := job.Steps[0]
	assert.NotNil(t, step)
	assert.EqualValues(t, step.GetRange().Start.Line, 18)
}

func TestLocalExecutor(t *testing.T) {
	content := `version: 2.1

orbs:
  localorb:
    executors:
      localexecutor:
          docker:
              - image: cimg/node:<< parameters.tag >>
          parameters:
              tag:
                  default: 1.0.0
                  description: Specify the Terraform Docker image tag for the executor
                  type: string`
	executorKey := "localexecutor"
	doc := GetDocForTests(t, content, "localorb")
	assert.Contains(t, doc.Executors, executorKey)
	executor := doc.Executors[executorKey]
	assert.EqualValues(t, executor.GetRange().Start.Line, 5)

	// Test parameters
	parameterKey := "tag"
	parameters := executor.GetParameters()
	assert.Contains(t, parameters, parameterKey)
	parameter := parameters[parameterKey]
	assert.EqualValues(t, parameter.GetRange().Start.Line, 9)
}

func TestLocalCommand(t *testing.T) {
	content := `version: 2.1

orbs:
  localorb:
    commands:
      localcommand:
        parameters:
            name:
                default: "world"
                type: string
        steps:
          - run: echo "Hello << parameter.name >>"`
	doc := GetDocForTests(t, content, "localorb")
	commandKey := "localcommand"
	assert.Contains(t, doc.Commands, commandKey)
	command := doc.Commands[commandKey]
	assert.EqualValues(t, command.Range.Start.Line, 5)

	// Test parameter
	parameterKey := "name"
	assert.Contains(t, command.Parameters, parameterKey)
	parameter := command.Parameters[parameterKey]
	assert.EqualValues(t, parameter.GetRange().Start.Line, 7)

	// Test step
	assert.Len(t, command.Steps, 1)
	step := command.Steps[0]
	assert.NotNil(t, step)
	assert.EqualValues(t, step.GetRange().Start.Line, 11)
}

// func TestCompleteLocalOrbFile(t *testing.T) {
// 	content := `version: 2.1

//     orbs:
//       localorb:
//         commands:
//           localcommand:
//             steps:
//               - run: echo "Hello world"

//         jobs:
//           localjob:
//             executor: localexecutor
//             steps:
//               - localcommand

//         executors:
//           localexecutor:
//             docker:
//               - image: cimg/base:2020.01`
// 	doc := GetDocForTests(t, content, "localorb")

// 	// Test command
// 	commandKey := "localcommand"
// 	assert.Contains(t, doc.Commands, commandKey)

// 	// Test executor
// 	executorKey := "localexecutor"
// 	assert.Contains(t, doc.Executors, executorKey)

// 	// Test job
// 	jobKey := "localjob"
// 	assert.Contains(t, doc.Jobs, jobKey)
// }

func GetDocForTests(t *testing.T, content string, orbKey string) parser.YamlDocument {
	context := testHelpers.GetDefaultLsContext()
	doc, err := parser.ParseFromContent([]byte(content), context, uri.File(""), protocol.Position{})
	assert.Nil(t, err)
	orbInfo, err := doc.GetOrbInfoFromName(orbKey, utils.CreateCache())
	assert.Nil(t, err)
	return doc.FromOrbParsedAttributesToYamlDocument(orbInfo.OrbParsedAttributes)
}

func TestOrbInLocalOrb(t *testing.T) {
	content := `version: 2.1

orbs:
  local:
    commands:
      cmd:
        parameters:
          target:
            type: string
        steps:
          - run: echo << parameters.target >>
    jobs:
      job:
        docker:
          - image: cimg/node:21.6.1
        steps:
          - cmd:
              target: world


jobs:
  do:
    docker:
      - image: cimg/node:21.6.1
    steps:
      - local/cmd:
          target: world

workflows:
  act:
    jobs:
      - do
      - local/job`
	context := testHelpers.GetDefaultLsContext()
	doc, err := parser.ParseFromContent([]byte(content), context, uri.File(""), protocol.Position{})
	assert.Nil(t, err)
	assert.Len(t, *doc.Diagnostics, 0)
	val := validate.Validate{
		APIs: validate.ValidateAPIs{
			DockerHub: dockerhub.NewAPI(),
		},
		Diagnostics: &[]protocol.Diagnostic{},
		Cache:       utils.CreateCache(),
		Doc:         doc,
		Context:     context,
	}
	val.Validate()
	errorDiagnostics := []protocol.Diagnostic{}
	for _, d := range *val.Diagnostics {
		if d.Severity == protocol.DiagnosticSeverityError {
			errorDiagnostics = append(errorDiagnostics, d)
		}
	}
	assert.Len(t, errorDiagnostics, 0)
}
