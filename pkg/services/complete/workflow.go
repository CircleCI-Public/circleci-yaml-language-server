package complete

import (
	"fmt"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func (ch *CompletionHandler) completeWorkflows() {
	wf, err := findWorkflow(ch.Params.Position, ch.Doc)
	if err != nil {
		return
	}

	if isJobInvocation(ch.Params.Position, wf) {
		ch.addJobsAndOrbsCompletion()
		return
	}

	if isInRequired(ch.Params.Position, wf) {
		ch.addExistingJobInvocations(wf)
		return
	}

	if wf.JobInvocations == nil {
		ch.addCompletionItemFieldWithNewLine("jobs")
	}
	if !wf.HasTrigger {
		ch.addCompletionItemFieldWithNewLine("trigger")
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

func (ch *CompletionHandler) addExistingJobInvocations(wf ast.Workflow) {
	for _, jobInvocation := range wf.JobInvocations {
		ch.addCompletionItem(jobInvocation.JobName)
	}
}

func findWorkflow(pos protocol.Position, doc yamlparser.YamlDocument) (ast.Workflow, error) {
	for _, wf := range doc.Workflows {
		if utils.PosInRange(wf.Range, pos) {
			return wf, nil
		}
	}
	return ast.Workflow{}, fmt.Errorf("no workflow found")
}

func isInRequired(pos protocol.Position, wf ast.Workflow) bool {
	for _, jobInvocation := range wf.JobInvocations {
		for _, require := range jobInvocation.Requires {
			if utils.PosInRange(require.Range, pos) {
				return true
			}
		}
	}

	return false
}

func isJobInvocation(pos protocol.Position, wf ast.Workflow) bool {
	if utils.PosInRange(wf.Range, pos) {
		jobInvocation := findJobInvocation(pos, wf)
		return jobInvocation != nil
	}
	return false
}

func findJobInvocation(pos protocol.Position, wf ast.Workflow) *ast.JobInvocation {
	for _, job := range wf.JobInvocations {
		if utils.PosInRange(job.JobNameRange, pos) {
			return &job
		}
	}
	return nil
}
