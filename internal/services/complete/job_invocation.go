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

// isInPreOrPostSteps says whether the position is in an invocation's
// pre-steps or post-steps.
func isInPreOrPostSteps(pos protocol.Position, jobInvocations []ast.JobInvocation) bool {
	for _, jobInvocation := range jobInvocations {
		for _, rng := range []protocol.Range{jobInvocation.PreStepsRange, jobInvocation.PostStepsRange} {
			if !position.IsDefaultRange(rng) && position.InRange(rng, pos) {
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
	if invocation := invocationNamedOn(parent, invocations); invocation != nil {
		return invocation, parent
	}
	return nil, 0
}

// invocationNamedOn is the invocation whose job's name is on a line.
func invocationNamedOn(line int, invocations []ast.JobInvocation) *ast.JobInvocation {
	for i := range invocations {
		if int(invocations[i].JobNameRange.Start.Line) == line {
			return &invocations[i]
		}
	}
	return nil
}

// completeJobInvocationBody offers the keys an invocation doesn't have yet:
// an invocation's own keys, and the parameters of the job it runs.
func (ch *CompletionHandler) completeJobInvocationBody(invocation *ast.JobInvocation, nameLine int) {
	keys := append(slices.Clone(jobInvocationKeys), ch.jobParameterNames(invocation)...)

	present := ch.stepBodyKeys(nameLine)
	for _, key := range keys {
		if !present[key] {
			ch.addCompletionItemField(key)
		}
	}
}

// jobParameterNames are the sorted names of the parameters of the job an
// invocation runs, of which an approval job has none.
func (ch *CompletionHandler) jobParameterNames(invocation *ast.JobInvocation) []string {
	if invocation.Type == "approval" {
		return nil
	}

	var params []string
	for param := range ch.Doc.GetDefinedParams(invocation.JobName, yamlparser.JobEntity, ch.Cache) {
		params = append(params, param)
	}
	slices.Sort(params)
	return params
}
