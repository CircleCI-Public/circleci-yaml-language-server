package validate

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

func (val Validate) ValidateJobGroups() {
	for _, jobGroup := range val.Doc.JobGroups {
		val.validateSingleJobGroup(jobGroup)
	}
}

func (val Validate) validateSingleJobGroup(jobGroup ast.JobGroup) {
	val.validateInvocations(jobGroup.JobInvocations, InvocationContext{Kind: InJobGroup, JobGroupName: jobGroup.Name})
	val.validateDAG(jobGroup.JobInvocations, jobGroup.JobsDAG)

	if !val.isJobGroupUsedInWorkflows(jobGroup.Name) {
		val.addDiagnostic(utils.CreateWarningDiagnosticFromRange(jobGroup.NameRange, "Job group is unused"))
	}
}
