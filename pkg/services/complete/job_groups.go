package complete

import (
	"fmt"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func (ch *CompletionHandler) completeJobGroups() {
	jobGroup, err := findJobGroup(ch.Params.Position, ch.Doc)
	if err != nil {
		return
	}

	if jobGroup.JobInvocations == nil {
		ch.addCompletionItemFieldWithNewLine("jobs")
	}

	if isJobInvocation(ch.Params.Position, jobGroup.JobInvocations) {
		ch.addJobsAndOrbsCompletion()
		// Unlike in workflows, we don't add completion items for job-groups here since nested job groups are not allowed
		return
	}

	// Add all the job invocations in the current job-group as completion items for "requires"
	// NOTE: Since this is called inside the top-level job-group key, this function will incorrectly suggest the user can
	// use other job-group invocations that are used within this group as valid options for requires.
	// This is acceptable as we have another error diagnostic indicating nested job-groups are not allowed.
	if isInRequires(ch.Params.Position, jobGroup.JobInvocations) {
		ch.addExistingJobInvocations(jobGroup.JobInvocations)
		return
	}
}

func findJobGroup(pos protocol.Position, doc yamlparser.YamlDocument) (ast.JobGroup, error) {
	for _, jobGroup := range doc.JobGroups {
		if utils.PosInRange(jobGroup.Range, pos) {
			return jobGroup, nil
		}
	}
	return ast.JobGroup{}, fmt.Errorf("no job group found")
}

func (ch *CompletionHandler) addJobGroupsCompletion() {
	for _, jobGroup := range ch.Doc.JobGroups {
		ch.addCompletionItem(jobGroup.Name)
	}
}
