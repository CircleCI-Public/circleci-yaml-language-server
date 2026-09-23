package validate

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
)

var TerminalJobStatuses = []string{"success", "failed", "canceled", "not_run", "unauthorized"}

func (val Validate) ValidateWorkflows() {
	for _, workflow := range val.Doc.Workflows {
		val.validateSingleWorkflow(workflow)
	}
}

func (val Validate) validateSingleWorkflow(workflow ast.Workflow) {
	if workflow.HasMaxAutoReruns {
		if workflow.MaxAutoReruns < 1 {
			val.addDiagnostic(diagnostic.Error(workflow.MaxAutoRerunsRange, "Must be greater than or equal to 1"))
		} else if workflow.MaxAutoReruns > 5 {
			val.addDiagnostic(diagnostic.Error(workflow.MaxAutoRerunsRange, "Must be less than or equal to 5"))
		}
	}

	val.validateInvocations(workflow.JobInvocations, InvocationContext{Kind: InWorkflow})
	val.validateDAG(workflow.JobInvocations, workflow.JobsDAG)
}
