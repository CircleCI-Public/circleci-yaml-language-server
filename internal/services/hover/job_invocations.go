package hover

import (
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

// JobInvocation is the hover for the name of a job that a workflow or a job
// group runs, whether the config's own or an orb's: the job's description and
// parameters.
func JobInvocation(doc yamlparser.YamlDocument, c *cache.Cache, pos protocol.Position) (string, bool) {
	name, ok := invokedJobAt(doc, pos)
	if !ok {
		return "", false
	}

	if job, ok := doc.Jobs[name]; ok {
		return describe(name, "job", job.Description, job.Parameters), true
	}
	if orb, jobName, ok := orbOf(doc, c, name); ok {
		if job, ok := orb.Jobs[jobName]; ok {
			return describe(name, "job", job.Description, job.Parameters), true
		}
	}
	return "", false
}

// invokedJobAt is the name of the job whose invocation, in a workflow or a job
// group, the cursor is on the name of.
func invokedJobAt(doc yamlparser.YamlDocument, pos protocol.Position) (string, bool) {
	for _, workflow := range doc.Workflows {
		for _, invocation := range workflow.JobInvocations {
			if position.InRange(invocation.JobNameRange, pos) {
				return invocation.JobName, true
			}
		}
	}
	for _, group := range doc.JobGroups {
		for _, invocation := range group.JobInvocations {
			if position.InRange(invocation.JobNameRange, pos) {
				return invocation.JobName, true
			}
		}
	}
	return "", false
}
