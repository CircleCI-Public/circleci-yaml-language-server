package parser_test

import (
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/dockerhub"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser/validate"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/testHelpers"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
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
	assert.Check(t, cmp.Contains(doc.Jobs, jobKey))
	job := doc.Jobs[jobKey]
	jobLine := job.Range.Start.Line
	assert.Check(t, cmp.Equal(jobLine, uint32(5)))

	// Test parameter
	parameterKey := "name"
	assert.Check(t, cmp.Contains(job.Parameters, parameterKey))
	parameter := job.Parameters[parameterKey]
	parameterLine := parameter.GetRange().Start.Line
	assert.Check(t, cmp.Equal(parameterLine, uint32(7)))

	// Test docker
	assert.Check(t, cmp.Equal(job.Docker.Name, "docker"))
	assert.Check(t, cmp.Len(job.Docker.Image, 1))
	image := job.Docker.Image[0]
	imageLine := image.ImageRange.Start.Line
	assert.Check(t, cmp.Equal(imageLine, uint32(11)))

	// Test step
	assert.Check(t, cmp.Len(job.Steps, 1))
	step := job.Steps[0]
	assert.Check(t, step != nil)
	stepLine := step.GetRange().Start.Line
	assert.Check(t, cmp.Equal(stepLine, uint32(13)))
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
	assert.Check(t, cmp.Contains(doc.Jobs, jobKey))
	job := doc.Jobs[jobKey]
	jobLine := job.Range.Start.Line
	assert.Check(t, cmp.Equal(jobLine, uint32(10)))

	// Test parameter
	parameterKey := "name"
	assert.Check(t, cmp.Contains(job.Parameters, parameterKey))
	parameter := job.Parameters[parameterKey]
	parameterLine := parameter.GetRange().Start.Line
	assert.Check(t, cmp.Equal(parameterLine, uint32(12)))

	// Test docker
	assert.Check(t, cmp.Equal(job.Docker.Name, "docker"))
	assert.Check(t, cmp.Len(job.Docker.Image, 1))
	image := job.Docker.Image[0]
	imageLine := image.ImageRange.Start.Line
	assert.Check(t, cmp.Equal(imageLine, uint32(16)))

	// Test step
	assert.Check(t, cmp.Len(job.Steps, 1))
	step := job.Steps[0]
	assert.Check(t, step != nil)
	stepLine := step.GetRange().Start.Line
	assert.Check(t, cmp.Equal(stepLine, uint32(18)))
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
	assert.Check(t, cmp.Contains(doc.Executors, executorKey))
	executor := doc.Executors[executorKey]
	executorLine := executor.GetRange().Start.Line
	assert.Check(t, cmp.Equal(executorLine, uint32(5)))

	// Test parameters
	parameterKey := "tag"
	parameters := executor.GetParameters()
	assert.Check(t, cmp.Contains(parameters, parameterKey))
	parameter := parameters[parameterKey]
	parameterLine := parameter.GetRange().Start.Line
	assert.Check(t, cmp.Equal(parameterLine, uint32(9)))
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
	assert.Check(t, cmp.Contains(doc.Commands, commandKey))
	command := doc.Commands[commandKey]
	commandLine := command.Range.Start.Line
	assert.Check(t, cmp.Equal(commandLine, uint32(5)))

	// Test parameter
	parameterKey := "name"
	assert.Check(t, cmp.Contains(command.Parameters, parameterKey))
	parameter := command.Parameters[parameterKey]
	parameterLine := parameter.GetRange().Start.Line
	assert.Check(t, cmp.Equal(parameterLine, uint32(7)))

	// Test step
	assert.Check(t, cmp.Len(command.Steps, 1))
	step := command.Steps[0]
	assert.Check(t, step != nil)
	stepLine := step.GetRange().Start.Line
	assert.Check(t, cmp.Equal(stepLine, uint32(11)))
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
	assert.Check(t, err)
	orbInfo, err := doc.GetOrbInfoFromName(orbKey, utils.CreateCache())
	assert.Check(t, err)
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
	assert.Check(t, err)
	assert.Check(t, cmp.Len(*doc.Diagnostics, 0))
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
	assert.Check(t, cmp.Len(errorDiagnostics, 0))
}
