package parser

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

func (doc *YamlDocument) assignContexts() {
	for _, workflow := range doc.Workflows {
		for _, jobRef := range workflow.JobRefs {
			for _, context := range jobRef.Context {
				if !doc.DoesJobExist(jobRef.JobName) {
					continue
				}
				job := doc.Jobs[jobRef.JobName]
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

func (doc *YamlDocument) addContextToJob(job ast.Job, context string) {
	if utils.FindInArray(*job.Contexts, context) < 0 {
		*job.Contexts = append(*job.Contexts, context)
	}
}

func (doc *YamlDocument) addContextToCommand(command ast.Command, context string) {
	if utils.FindInArray(*command.Contexts, context) < 0 {
		*command.Contexts = append(*command.Contexts, context)
	}
}
