package validate

import (
	"context"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
)

func (val Validate) ValidateJobGroups(ctx context.Context) {
	for _, jobGroup := range val.Doc.JobGroups {
		val.validateSingleJobGroup(ctx, jobGroup)
	}
}

func (val Validate) validateSingleJobGroup(ctx context.Context, jobGroup ast.JobGroup) {
	val.validateInvocations(ctx, jobGroup.JobInvocations, InvocationContext{Kind: InJobGroup, JobGroupName: jobGroup.Name})
	val.validateDAG(jobGroup.JobInvocations, jobGroup.JobsDAG)

	if !val.isJobGroupUsedInWorkflows(jobGroup.Name) {
		val.addDiagnostic(diagnostic.Warning(jobGroup.NameRange, "Job group is unused"))
	}
}
