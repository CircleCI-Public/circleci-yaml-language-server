package complete

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func isJobInvocation(pos protocol.Position, invocations []ast.JobInvocation) bool {
	jobInvocation := findJobInvocation(pos, invocations)
	return jobInvocation != nil
}

func findJobInvocation(pos protocol.Position, invocations []ast.JobInvocation) *ast.JobInvocation {
	for _, jobInvocation := range invocations {
		if utils.PosInRange(jobInvocation.JobNameRange, pos) {
			return &jobInvocation
		}
	}
	return nil
}

func isInRequires(pos protocol.Position, jobInvocations []ast.JobInvocation) bool {
	for _, jobInvocation := range jobInvocations {
		for _, require := range jobInvocation.Requires {
			if utils.PosInRange(require.Range, pos) {
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
