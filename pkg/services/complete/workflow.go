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

	if isJobReference(ch.Params.Position, wf) {
		ch.addJobsAndOrbsCompletion()
		return
	}

	if isInRequired(ch.Params.Position, wf) {
		ch.addExistingJobReferences(wf)
		return
	}

	if wf.JobRefs == nil {
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

func (ch *CompletionHandler) addExistingJobReferences(wf ast.Workflow) {
	for _, jobRef := range wf.JobRefs {
		ch.addCompletionItem(jobRef.JobName)
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
	for _, jobRef := range wf.JobRefs {
		for _, require := range jobRef.Requires {
			if utils.PosInRange(require.Range, pos) {
				return true
			}
		}
	}

	return false
}

func isJobReference(pos protocol.Position, wf ast.Workflow) bool {
	if utils.PosInRange(wf.Range, pos) {
		jobRef := findJobRef(pos, wf)
		return jobRef != nil
	}
	return false
}

func findJobRef(pos protocol.Position, wf ast.Workflow) *ast.JobRef {
	for _, job := range wf.JobRefs {
		if utils.PosInRange(job.JobNameRange, pos) {
			return &job
		}
	}
	return nil
}
