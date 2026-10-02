package complete

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"go.lsp.dev/protocol"

	ast2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func (ch *CompletionHandler) completeJobs(ctx context.Context) {
	job, err := findJob(ch.Params.Position, ch.Doc)
	if err != nil {
		return
	}

	if key, lines, parent := ch.valueAt(); parent != -1 && !position.IsDefaultRange(job.ExecutorRange) &&
		parent == int(job.ExecutorRange.Start.Line) && executorMapping.MatchString(lines[parent]) {
		if param, ok := ch.executorParameters(ctx, job.Executor)[key]; ok {
			ch.addParameterValues(param)
			return
		}
	}

	if lines, parent := ch.keyParent(); parent != -1 && !position.IsDefaultRange(job.ExecutorRange) &&
		parent == int(job.ExecutorRange.Start.Line) && executorMapping.MatchString(lines[parent]) {
		ch.completeExecutorMapping(ctx, job.Executor, parent)
		return
	}

	if ch.completeJobExecutor(ctx, job) || ch.completeDockerEntry() || ch.completeReleaseValidation(job) {
		return
	}

	switch true {
	case position.InRange(job.ExecutorRange, ch.Params.Position):
		ch.addExecutorsCompletion(ctx)
		return
	case position.InRange(job.ParametersRange, ch.Params.Position):
		ch.addParametersDefinitionCompletion(ctx, job.Parameters)
		return
	case position.InRange(job.StepsRange, ch.Params.Position):
		ch.completeSteps(ctx, job.Name, true, ch.nodeToComplete())
		return
	case position.InRange(job.DockerRange, ch.Params.Position):
		ch.completeDockerExecutor(ctx, job.Docker)
		return
	case position.InRange(job.TypeRange, ch.Params.Position):
		ch.addJobTypeCompletion()
		return
	}

	if _, parent := ch.keyParent(); parent == startLine(job.NameRange) {
		ch.Items = append(ch.Items, (*job.CompletionItem)...)
	}
}

func (ch *CompletionHandler) orbsJobs(ctx context.Context) {
	for _, orb := range ch.Doc.Orbs {
		// Local orbs jobs are added directly within ch.Doc.Jobs
		orbInfo := ch.GetOrbInfo(ctx, orb)
		if orbInfo != nil {
			for jobName := range orbInfo.Jobs {
				jobName = fmt.Sprintf("%s/%s", orb.Name, jobName)
				ch.addCompletionItem(jobName)
			}
		}
	}
}

func (ch *CompletionHandler) addExecutorsCompletion(ctx context.Context) {
	for _, executor := range ch.Doc.Executors {
		ch.addCompletionItem(executor.GetName())
	}
	for _, alias := range ch.Doc.Aliases.Executors {
		ch.addCompletionItem(alias.Name)
	}

	for _, orb := range ch.Doc.Orbs {
		executor := ch.getOrbExecutors(ctx, orb)
		for _, executor := range executor {
			ch.addCompletionItem(fmt.Sprintf("%s/%s", orb.Name, executor.GetName()))
		}
	}
}

var executorMapping = regexp.MustCompile(`^\s*executor\s*:\s*$`)

// completeExecutorMapping offers the keys an executor given as a mapping
// doesn't have yet: its name, and the parameters the executor declares.
func (ch *CompletionHandler) completeExecutorMapping(ctx context.Context, name string, executorLine int) {
	keys := []string{"name"}
	params := []string{}
	for param := range ch.executorParameters(ctx, name) {
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
// declares, named directly or through an alias.
func (ch *CompletionHandler) executorParameters(ctx context.Context, name string) map[string]ast2.Parameter {
	if executor, ok := ch.Doc.ResolveExecutor(ctx, name, ch.Cache); ok {
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

var releaseValidationKeys = []string{"enabled", "evaluation_time", "auto_rollback_on_failure", "webhooks"}

// completeReleaseValidation offers the keys a release job's validation
// doesn't have yet, or the values of its booleans, and says whether the
// cursor is at one.
func (ch *CompletionHandler) completeReleaseValidation(job ast2.Job) bool {
	isValidation := func(lines []string, line int) bool {
		return line != -1 && strings.TrimSpace(lines[line]) == "validation:" && parentLine(lines, line) == startLine(job.NameRange)
	}

	if key, lines, parent := ch.valueAt(); isValidation(lines, parent) {
		if key == "enabled" || key == "auto_rollback_on_failure" {
			ch.addCompletionItem("true")
			ch.addCompletionItem("false")
		}
		return true
	}

	lines, parent := ch.keyParent()
	if !isValidation(lines, parent) {
		return false
	}
	present := ch.stepBodyKeys(parent)
	for _, key := range releaseValidationKeys {
		if !present[key] {
			ch.addCompletionItemField(key)
		}
	}
	return true
}
