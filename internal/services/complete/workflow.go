package complete

import (
	"fmt"
	"strings"

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

	ch.completeWorkflowKeys(wf)
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

// triggerMappingKeys are the keys of each mapping in a workflow's triggers,
// by the path to the mapping from the workflow.
var triggerMappingKeys = map[string][]string{
	"triggers.schedule":                  {"cron", "filters"},
	"triggers.schedule.filters":          {"branches"},
	"triggers.schedule.filters.branches": {"only", "ignore"},
}

// completeWorkflowKeys offers the keys the workflow, or a mapping in its
// triggers, doesn't have yet, when the cursor is at a key of one.
func (ch *CompletionHandler) completeWorkflowKeys(wf ast.Workflow) {
	lines, parent := ch.keyParent()
	if parent == -1 {
		ch.completeTriggerItem(wf, lines)
		return
	}

	path := []string{}
	for line := parent; line != startLine(wf.NameRange); line = parentLine(lines, line) {
		if line == -1 {
			return
		}
		match := mappingKey.FindStringSubmatch(lines[line])
		if match == nil {
			return
		}
		path = append([]string{match[1]}, path...)
	}

	if len(path) == 0 {
		ch.addWorkflowKeys(wf)
		return
	}
	present := ch.stepBodyKeys(parent)
	for _, key := range triggerMappingKeys[strings.Join(path, ".")] {
		if !present[key] {
			ch.addCompletionItemField(key)
		}
	}
}

// completeTriggerItem offers `schedule` for an item being written in a
// workflow's triggers.
func (ch *CompletionHandler) completeTriggerItem(wf ast.Workflow, lines []string) {
	pos := ch.Params.Position
	if int(pos.Line) >= len(lines) {
		return
	}
	before := lines[pos.Line][:min(int(pos.Character), len(lines[pos.Line]))]
	if !listItemBeingWritten.MatchString(before) {
		return
	}
	parent := lineAbove(lines, int(pos.Line), indentation(before))
	if parent != -1 && strings.TrimSpace(lines[parent]) == "triggers:" && parentLine(lines, parent) == startLine(wf.NameRange) {
		ch.addCompletionItemFieldWithNewLine("schedule")
	}
}

// addWorkflowKeys offers the keys a workflow doesn't have yet.
func (ch *CompletionHandler) addWorkflowKeys(wf ast.Workflow) {
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
