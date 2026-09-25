package complete

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

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

	if key, lines, parent := ch.valueAt(); parent != -1 && !position.IsDefaultRange(job.ExecutorRange) &&
		parent == int(job.ExecutorRange.Start.Line) && executorMapping.MatchString(lines[parent]) {
		if param, ok := ch.executorParameters(job.Executor)[key]; ok {
			ch.addParameterValues(param)
			return
		}
	}

	if lines, parent := ch.keyParent(); parent != -1 && !position.IsDefaultRange(job.ExecutorRange) &&
		parent == int(job.ExecutorRange.Start.Line) && executorMapping.MatchString(lines[parent]) {
		ch.completeExecutorMapping(job.Executor, parent)
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
		ch.completeSteps(job.Name, true, ch.nodeToComplete())
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

var executorMapping = regexp.MustCompile(`^\s*executor\s*:\s*$`)

// completeExecutorMapping offers the keys an executor given as a mapping
// doesn't have yet: its name, and the parameters the executor declares.
func (ch *CompletionHandler) completeExecutorMapping(name string, executorLine int) {
	keys := []string{"name"}
	params := []string{}
	for param := range ch.executorParameters(name) {
		params = append(params, param)
	}
	slices.Sort(params)
	keys = append(keys, params...)

	present := ch.stepBodyKeys(executorLine)
	for _, key := range keys {
		if !present[key] {
			ch.addCompletionItemField(key)
		}
	}
}

// executorParameters are the parameters a local, inline-orb or orb executor
// declares.
func (ch *CompletionHandler) executorParameters(name string) map[string]ast2.Parameter {
	if executor, ok := ch.Doc.Executors[name]; ok {
		return executor.GetParameters()
	}

	orbName, executorName, ok := strings.Cut(name, "/")
	if !ok {
		return nil
	}
	orbInfo, err := ch.Doc.GetOrbInfoFromName(orbName, ch.Cache)
	if err != nil || orbInfo == nil {
		return nil
	}
	if executor, ok := orbInfo.Executors[executorName]; ok {
		return executor.GetParameters()
	}
	return nil
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
