package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
)

func getWarningMessages(diags *[]protocol.Diagnostic) []string {
	var msgs []string
	for _, d := range *diags {
		if d.Severity == protocol.DiagnosticSeverityWarning {
			msgs = append(msgs, d.Message)
		}
	}
	return msgs
}

func TestCheckNames_NoConflicts(t *testing.T) {
	yaml := `
version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - checkout

commands:
  setup:
    steps:
      - checkout

workflows:
  main:
    jobs:
      - build
`
	val := CreateValidateFromYAML(yaml)
	val.CheckNames()
	assert.Empty(t, *val.Diagnostics)
}

func TestCheckNames_WorkflowJobConflict(t *testing.T) {
	yaml := `
version: 2.1

jobs:
  deploy:
    docker:
      - image: cimg/base:current
    steps:
      - checkout

workflows:
  deploy:
    jobs:
      - deploy
`
	val := CreateValidateFromYAML(yaml)
	val.CheckNames()

	msgs := getWarningMessages(val.Diagnostics)
	assert.Len(t, msgs, 2)
	assert.Contains(t, msgs[0], `"deploy"`)
	assert.Contains(t, msgs[0], "job")
	assert.Contains(t, msgs[1], `"deploy"`)
	assert.Contains(t, msgs[1], "workflow")
}

func TestCheckNames_WorkflowCommandConflict(t *testing.T) {
	yaml := `
version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - checkout

commands:
  ci:
    steps:
      - checkout

workflows:
  ci:
    jobs:
      - build
`
	val := CreateValidateFromYAML(yaml)
	val.CheckNames()

	msgs := getWarningMessages(val.Diagnostics)
	assert.Len(t, msgs, 2)
	assert.Contains(t, msgs[0], `"ci"`)
	assert.Contains(t, msgs[0], "command")
	assert.Contains(t, msgs[1], `"ci"`)
	assert.Contains(t, msgs[1], "workflow")
}

func TestCheckNames_JobCommandConflict(t *testing.T) {
	yaml := `
version: 2.1

jobs:
  setup:
    docker:
      - image: cimg/base:current
    steps:
      - checkout

commands:
  setup:
    steps:
      - checkout

workflows:
  main:
    jobs:
      - setup
`
	val := CreateValidateFromYAML(yaml)
	val.CheckNames()

	msgs := getWarningMessages(val.Diagnostics)
	assert.Len(t, msgs, 2)
	assert.Contains(t, msgs[0], `"setup"`)
	assert.Contains(t, msgs[0], "command")
	assert.Contains(t, msgs[1], `"setup"`)
	assert.Contains(t, msgs[1], "job")
}

func TestCheckNames_MultipleConflicts(t *testing.T) {
	yaml := `
version: 2.1

jobs:
  shared:
    docker:
      - image: cimg/base:current
    steps:
      - checkout

commands:
  shared:
    steps:
      - checkout

workflows:
  shared:
    jobs:
      - shared
`
	val := CreateValidateFromYAML(yaml)
	val.CheckNames()

	// 3 pairs: workflow-job, workflow-command, job-command = 6 warnings
	msgs := getWarningMessages(val.Diagnostics)
	assert.Len(t, msgs, 6)
	for _, msg := range msgs {
		assert.Contains(t, msg, `"shared"`)
	}
}

func TestCheckNames_SameKindNoDiagnostic(t *testing.T) {
	// Two jobs with different names, all is good in the hood
	yaml := `
version: 2.1

jobs:
  build:
    docker:
      - image: cimg/base:current
    steps:
      - checkout
  test:
    docker:
      - image: cimg/base:current
    steps:
      - checkout

workflows:
  main:
    jobs:
      - build
      - test
`
	val := CreateValidateFromYAML(yaml)
	val.CheckNames()
	assert.Empty(t, *val.Diagnostics)
}
