package complete

import (
	"fmt"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func (ch *CompletionHandler) completeJobs() {
	job, err := findJob(ch.Params.Position, ch.Doc)
	if err != nil {
		return
	}

	switch true {
	case utils.PosInRange(job.ExecutorRange, ch.Params.Position):
		ch.addExecutorsCompletion()
		return
	case utils.PosInRange(job.ParametersRange, ch.Params.Position):
		ch.addParametersDefinitionCompletion(job.Parameters)
		return
	case utils.PosInRange(job.StepsRange, ch.Params.Position):
		nodeToComplete, _, _ := utils.NodeAtPos(ch.Doc.RootNode, ch.Params.Position)
		if nodeToComplete.Type() == ":" {
			nodeToComplete = nodeToComplete.PrevSibling()
		}

		ch.completeSteps(true, nodeToComplete)
		return
	case utils.PosInRange(job.DockerRange, ch.Params.Position):
		ch.completeDockerExecutor(job.Docker)
		return
	}

	ch.Items = append(ch.Items, (*job.CompletionItem)...)
}

func (ch *CompletionHandler) orbsJobs() {
	for _, orb := range ch.Doc.Orbs {
		remoteOrb := ch.Cache.OrbCache.GetOrb(orb.Url.GetOrbID())
		if remoteOrb != nil {
			for jobName := range remoteOrb.Jobs {
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

func findJob(pos protocol.Position, doc yamlparser.YamlDocument) (ast.Job, error) {
	for _, job := range doc.Jobs {
		if utils.PosInRange(job.Range, pos) {
			return job, nil
		}
	}
	return ast.Job{}, fmt.Errorf("no job found")
}
