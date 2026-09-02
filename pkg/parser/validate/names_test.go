package validate

import (
	"testing"

	"go.lsp.dev/protocol"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
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
	assert.Check(t, cmp.Len(*val.Diagnostics, 0))
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
	assert.Check(t, cmp.Len(msgs, 2))
	assert.Check(t, cmp.Contains(msgs[0], `"deploy"`))
	assert.Check(t, cmp.Contains(msgs[0], "job"))
	assert.Check(t, cmp.Contains(msgs[1], `"deploy"`))
	assert.Check(t, cmp.Contains(msgs[1], "workflow"))
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
	assert.Check(t, cmp.Len(msgs, 2))
	assert.Check(t, cmp.Contains(msgs[0], `"ci"`))
	assert.Check(t, cmp.Contains(msgs[0], "command"))
	assert.Check(t, cmp.Contains(msgs[1], `"ci"`))
	assert.Check(t, cmp.Contains(msgs[1], "workflow"))
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
	assert.Check(t, cmp.Len(msgs, 2))
	assert.Check(t, cmp.Contains(msgs[0], `"setup"`))
	assert.Check(t, cmp.Contains(msgs[0], "command"))
	assert.Check(t, cmp.Contains(msgs[1], `"setup"`))
	assert.Check(t, cmp.Contains(msgs[1], "job"))
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
	assert.Check(t, cmp.Len(msgs, 6))
	for _, msg := range msgs {
		assert.Check(t, cmp.Contains(msg, `"shared"`))
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
	assert.Check(t, cmp.Len(*val.Diagnostics, 0))
}
