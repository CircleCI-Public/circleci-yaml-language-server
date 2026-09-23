package parser

import (
	"slices"

	ast2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
)

func (doc *YamlDocument) assignContexts() {
	for _, workflow := range doc.Workflows {
		for _, jobInvocation := range workflow.JobInvocations {
			for _, context := range jobInvocation.Context {
				if !doc.DoesJobExist(jobInvocation.JobName) {
					continue
				}
				job := doc.Jobs[jobInvocation.JobName]
				doc.addContextToJob(job, context.Text)
				for _, step := range job.Steps {
					if doc.DoesCommandExist(step.GetName()) {
						command := doc.Commands[step.GetName()]
						doc.addContextToCommand(command, context.Text)
					} else if doc.DoesJobExist(step.GetName()) {
						job := doc.Jobs[step.GetName()]
						doc.addContextToJob(job, context.Text)
					}
				}
			}
		}
	}
}

func (doc *YamlDocument) addContextToJob(job ast2.Job, context string) {
	if !slices.Contains(*job.Contexts, context) {
		*job.Contexts = append(*job.Contexts, context)
	}
}

func (doc *YamlDocument) addContextToCommand(command ast2.Command, context string) {
	if !slices.Contains(*command.Contexts, context) {
		*command.Contexts = append(*command.Contexts, context)
	}
}
