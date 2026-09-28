package validate

import (
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
)

var TerminalJobStatuses = []string{"success", "failed", "canceled", "not_run", "unauthorized"}

func (val Validate) ValidateWorkflows() {
	for _, workflow := range val.Doc.Workflows {
		val.validateSingleWorkflow(workflow)
	}
	val.validateImplicitWorkflow()
}

// orbOnlyKeys are top-level keys an orb has and a config doesn't.
var orbOnlyKeys = []string{"description", "display", "examples"}

// validateImplicitWorkflow reports a config with no workflows whose implicit
// workflow would run a job named build that doesn't exist. An orb has no
// workflows to run, and broken YAML may hide the workflows it has, so
// neither is reported.
func (val Validate) validateImplicitWorkflow() {
	if !val.Doc.HasNoWorkflows() || val.isOrb() || len(val.Doc.SyntaxErrors()) > 0 {
		return
	}
	if _, ok := val.Doc.Jobs["build"]; ok {
		return
	}
	if _, ok := val.Doc.Aliases.Jobs["build"]; ok {
		return
	}

	rng := protocol.Range{}
	for _, key := range []string{"workflows", "jobs", "version"} {
		if keyRange, ok := val.Doc.SectionKeyRanges[key]; ok {
			rng = keyRange
			break
		}
	}
	val.addDiagnostic(diagnostic.Error(rng, "There are no workflows or build jobs in the config."))
}

// isOrb reports whether the document is an orb rather than a config: one
// with a key only orbs have, or the source of a remote orb.
func (val Validate) isOrb() bool {
	for _, key := range orbOnlyKeys {
		if _, ok := val.Doc.SectionKeyRanges[key]; ok {
			return true
		}
	}
	if val.Cache == nil {
		return false
	}
	_, ok := val.Cache.OrbIDOfSource(val.Doc.URI.FsPath())
	return ok
}

func (val Validate) validateSingleWorkflow(workflow ast.Workflow) {
	if workflow.HasMaxAutoReruns {
		if workflow.MaxAutoReruns < 1 {
			val.addDiagnostic(diagnostic.Error(workflow.MaxAutoRerunsRange, "Must be greater than or equal to 1"))
		} else if workflow.MaxAutoReruns > 5 {
			val.addDiagnostic(diagnostic.Error(workflow.MaxAutoRerunsRange, "Must be less than or equal to 5"))
		}
	}

	val.validateInvocations(workflow.JobInvocations, InvocationContext{Kind: InWorkflow, WorkflowName: workflow.Name})
	val.validateDAG(workflow.JobInvocations, workflow.JobsDAG)
}
