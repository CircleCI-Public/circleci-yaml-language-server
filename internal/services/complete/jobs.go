package complete

import (
	"fmt"

	"go.lsp.dev/protocol"

	ast2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func (ch *CompletionHandler) completeJobs() {
	job, err := findJob(ch.Params.Position, ch.Doc)
	if err != nil {
		return
	}

	switch true {
	case position.InRange(job.ExecutorRange, ch.Params.Position):
		ch.addExecutorsCompletion()
		return
	case position.InRange(job.ParametersRange, ch.Params.Position):
		ch.addParametersDefinitionCompletion(job.Parameters)
		return
	case position.InRange(job.StepsRange, ch.Params.Position):
		nodeToComplete, _, _ := position.NodeAt(ch.Doc.RootNode, ch.Params.Position)
		if nodeToComplete.Kind() == ":" {
			nodeToComplete = nodeToComplete.PrevSibling()
		}

		ch.completeSteps(job.Name, true, true, nodeToComplete)
		return
	case position.InRange(job.DockerRange, ch.Params.Position):
		ch.completeDockerExecutor(job.Docker)
		return
	case position.InRange(job.TypeRange, ch.Params.Position):
		ch.addJobTypeCompletion()
		return
	}

	ch.Items = append(ch.Items, (*job.CompletionItem)...)
}

func (ch *CompletionHandler) orbsJobs() {
	for _, orb := range ch.Doc.Orbs {
		// Local orbs jobs are added directly within ch.Doc.Jobs
		orbInfo := ch.GetOrbInfo(orb)
		if orbInfo != nil {
			for jobName := range orbInfo.Jobs {
				jobName = fmt.Sprintf("%s/%s", orb.Name, jobName)
				ch.addCompletionItem(jobName)
			}
		}
	}
}

func (ch *CompletionHandler) userDefinedJobs() {
	for _, job := range ch.Doc.Jobs {
		ch.addCompletionItem(job.Name)
	}
}

func (ch *CompletionHandler) addExecutorsCompletion() {
	for _, executor := range ch.Doc.Executors {
		ch.addCompletionItem(executor.GetName())
	}

	for _, orb := range ch.Doc.Orbs {
		executor := ch.getOrbExecutors(orb)
		for _, executor := range executor {
			ch.addCompletionItem(fmt.Sprintf("%s/%s", orb.Name, executor.GetName()))
		}
	}
}

func findJob(pos protocol.Position, doc yamlparser.YamlDocument) (ast2.Job, error) {
	for _, job := range doc.Jobs {
		if position.InRange(job.Range, pos) {
			return job, nil
		}
	}
	return ast2.Job{}, fmt.Errorf("no job found")
}

func (ch *CompletionHandler) addJobTypeCompletion() {
	for _, jobType := range ast2.JobTypes {
		ch.addCompletionItem(jobType)
	}
}
