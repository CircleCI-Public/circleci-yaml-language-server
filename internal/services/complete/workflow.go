package complete

import (
	"fmt"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func (ch *CompletionHandler) completeWorkflows() {
	wf, err := findWorkflow(ch.Params.Position, ch.Doc)
	if err != nil {
		return
	}

	if isJobInvocation(ch.Params.Position, wf.JobInvocations) {
		ch.addJobsAndOrbsCompletion()
		ch.addJobGroupsCompletion()
		return
	}

	if ch.completeInJobInvocations(wf.JobInvocations) {
		return
	}

	ch.addWorkflowKeys(wf)
}

// completeInJobInvocations completes inside the body of one of a workflow's
// or job group's job invocations, and says whether the cursor was in one.
func (ch *CompletionHandler) completeInJobInvocations(invocations []ast.JobInvocation) bool {
	if ch.completeContextName(invocations) {
		return true
	}

	if ch.completeRequiredStatus() {
		return true
	}

	if isInRequires(ch.Params.Position, invocations) {
		ch.addExistingJobInvocations(invocations)
		return true
	}

	if isInPreOrPostSteps(ch.Params.Position, invocations) {
		ch.completeStepList(ch.nodeToComplete())
		return true
	}

	if key, lines, parent := ch.valueAt(); parent != -1 && stepWithBody.MatchString(lines[parent]) {
		if invocation := invocationNamedOn(parent, invocations); invocation != nil {
			params := ch.Doc.GetDefinedParams(invocation.JobName, yamlparser.JobEntity, ch.Cache)
			if param, ok := params[key]; ok {
				ch.addParameterValues(param)
			}
			return true
		}
	}

	if ch.completeInvocationMapping(invocations) {
		return true
	}

	if invocation, nameLine := ch.jobInvocationBodyAt(invocations); invocation != nil {
		ch.completeJobInvocationBody(invocation, nameLine)
		return true
	}

	return false
}

// addWorkflowKeys offers the keys a workflow doesn't have yet, when the
// cursor is at a workflow's own keys rather than inside one of them.
func (ch *CompletionHandler) addWorkflowKeys(wf ast.Workflow) {
	pos := ch.Params.Position
	for _, rng := range []protocol.Range{wf.NameRange, wf.JobsRange, wf.TriggersRange, wf.WhenRange, wf.UnlessRange, wf.MaxAutoRerunsRange} {
		if !position.IsDefaultRange(rng) && position.InRange(rng, pos) {
			return
		}
	}

	if position.IsDefaultRange(wf.JobsRange) {
		ch.addCompletionItemFieldWithNewLine("jobs")
	}
	if !wf.HasTrigger {
		ch.addCompletionItemFieldWithNewLine("triggers")
	}
	if position.IsDefaultRange(wf.WhenRange) {
		ch.addCompletionItemField("when")
	}
	if position.IsDefaultRange(wf.UnlessRange) {
		ch.addCompletionItemField("unless")
	}
	if position.IsDefaultRange(wf.MaxAutoRerunsRange) {
		ch.addCompletionItemField("max_auto_reruns")
	}
}

func findWorkflow(pos protocol.Position, doc yamlparser.YamlDocument) (ast.Workflow, error) {
	for _, wf := range doc.Workflows {
		if position.InRange(wf.Range, pos) {
			return wf, nil
		}
	}
	return ast.Workflow{}, fmt.Errorf("no workflow found")
}
