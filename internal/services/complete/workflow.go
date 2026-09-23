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

	if wf.JobInvocations == nil {
		ch.addCompletionItemFieldWithNewLine("jobs")
	}

	if isJobInvocation(ch.Params.Position, wf.JobInvocations) {
		ch.addJobsAndOrbsCompletion()
		ch.addJobGroupsCompletion()
		return
	}

	// Add all the job/job-group invocations in the current workflow as completion items for "requires"
	if isInRequires(ch.Params.Position, wf.JobInvocations) {
		ch.addExistingJobInvocations(wf.JobInvocations)
		return
	}

	if !wf.HasTrigger {
		ch.addCompletionItemFieldWithNewLine("trigger")
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
