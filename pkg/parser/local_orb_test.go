package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
	doc, err := GetParsedYAMLWithContent([]byte(content))
	assert.Nil(t, err)

	// Test job
	jobKey := "localorb/localjob"
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
	doc, err := GetParsedYAMLWithContent([]byte(content))
	assert.Nil(t, err)

	// Test job
	jobKey := "localorb/localjob"
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
	doc, err := GetParsedYAMLWithContent([]byte(content))
	assert.Nil(t, err)

	// Test executor
	executorKey := "localorb/localexecutor"
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
	doc, err := GetParsedYAMLWithContent([]byte(content))
	assert.Nil(t, err)

	// Test command
	commandKey := "localorb/localcommand"
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

func TestCompleteLocalOrbFile(t *testing.T) {
	content := `version: 2.1

orbs:
  localorb:
    jobs:
      localjob:
        executor: localexecutor
        steps:
          - localcommand

    executors:
      localexecutor:
        docker:
          - image: cimg/base:2020.01

    commands:
      localcommand:
        steps:
          - run: echo "Hello world"

workflows:
  someworkflow:
    jobs:
      - localorb/localjob`
	doc, err := GetParsedYAMLWithContent([]byte(content))
	assert.Nil(t, err)

	assert.Len(t, *doc.Diagnostics, 0)

	// Test command
	commandKey := "localorb/localcommand"
	assert.Contains(t, doc.Commands, commandKey)

	// Test executor
	executorKey := "localorb/localexecutor"
	assert.Contains(t, doc.Executors, executorKey)

	// Test job
	jobKey := "localorb/localjob"
	assert.Contains(t, doc.Jobs, jobKey)
}
