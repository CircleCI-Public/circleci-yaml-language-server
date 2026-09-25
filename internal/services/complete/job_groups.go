package complete

import (
	"fmt"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
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

	// For requires, this offers job-group invocations used within this group
	// too, though nested job groups are reported as not allowed.
	ch.completeInJobInvocations(jobGroup.JobInvocations)
}

func findJobGroup(pos protocol.Position, doc yamlparser.YamlDocument) (ast.JobGroup, error) {
	for _, jobGroup := range doc.JobGroups {
		if position.InRange(jobGroup.Range, pos) {
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
