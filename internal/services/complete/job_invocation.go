package complete

import (
	"slices"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func isJobInvocation(pos protocol.Position, invocations []ast.JobInvocation) bool {
	jobInvocation := findJobInvocation(pos, invocations)
	return jobInvocation != nil
}

func findJobInvocation(pos protocol.Position, invocations []ast.JobInvocation) *ast.JobInvocation {
	for _, jobInvocation := range invocations {
		if position.InRange(jobInvocation.JobNameRange, pos) {
			return &jobInvocation
		}
	}
	return nil
}

func isInRequires(pos protocol.Position, jobInvocations []ast.JobInvocation) bool {
	for _, jobInvocation := range jobInvocations {
		for _, require := range jobInvocation.Requires {
			if position.InRange(require.Range, pos) {
				return true
			}
		}
	}

	return false
}

// addExistingJobInvocations adds a completion item for each invocation provided.
// It uses the `name:` override, because that's how other jobs reference this invocation in requires.
func (ch *CompletionHandler) addExistingJobInvocations(jobInvocations []ast.JobInvocation) {
	for _, jobInvocation := range jobInvocations {
		ch.addCompletionItem(jobInvocation.StepName)
	}
}

func (ch *CompletionHandler) addJobsAndOrbsCompletion() {
	ch.addJobsCompletion()
	ch.orbsJobs()
}

func (ch *CompletionHandler) addJobsCompletion() {
	for _, job := range ch.Doc.Jobs {
		ch.addCompletionItem(job.Name)
	}
}

// jobInvocationKeys are the keys a workflow's job invocation takes.
var jobInvocationKeys = []string{
	"requires", "context", "filters", "matrix", "name", "type",
	"pre-steps", "post-steps", "serial-group", "override-with",
}

// jobInvocationBodyAt is the invocation whose body the cursor is at a key
// of, and the line its name is on.
func (ch *CompletionHandler) jobInvocationBodyAt(invocations []ast.JobInvocation) (*ast.JobInvocation, int) {
	lines, parent := ch.keyParent()
	if parent == -1 || !stepWithBody.MatchString(lines[parent]) {
		return nil, 0
	}
	for i := range invocations {
		if int(invocations[i].JobNameRange.Start.Line) == parent {
			return &invocations[i], parent
		}
	}
	return nil, 0
}

// completeJobInvocationBody offers the keys an invocation doesn't have yet:
// an invocation's own keys, and the parameters of the job it runs.
func (ch *CompletionHandler) completeJobInvocationBody(invocation *ast.JobInvocation, nameLine int) {
	keys := slices.Clone(jobInvocationKeys)

	if invocation.Type != "approval" {
		var params []string
		for param := range ch.Doc.GetDefinedParams(invocation.JobName, yamlparser.JobEntity, ch.Cache) {
			params = append(params, param)
		}
		slices.Sort(params)
		keys = append(keys, params...)
	}

	present := ch.stepBodyKeys(nameLine)
	for _, key := range keys {
		if !present[key] {
			ch.addCompletionItemField(key)
		}
	}
}
