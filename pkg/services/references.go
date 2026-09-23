package languageservice

import (
	"fmt"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/paramref"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"go.lsp.dev/protocol"
)

func References(params protocol.ReferenceParams, cache *cache.Cache, context *session.Settings) ([]protocol.Location, error) {
	yamlDocument, err := yamlparser.ParseFromUriWithCache(params.TextDocument.URI, cache, context)

	if err != nil {
		return nil, err
	}
	defer yamlDocument.Close()

	ref := ReferenceHandler{
		Doc:        yamlDocument,
		Params:     params,
		Cache:      cache,
		FoundSteps: &[]StepRangeAndName{},
	}

	return ref.GetReferences()
}

type ReferenceHandler struct {
	Doc        yamlparser.YamlDocument
	Params     protocol.ReferenceParams
	Cache      *cache.Cache
	FoundSteps *[]StepRangeAndName
}

func (ref ReferenceHandler) GetReferences() ([]protocol.Location, error) {
	cmdName := ""
	isOrb := false

	if position.InRange(ref.Doc.OrbsRange, ref.Params.Position) {
		var orb ast.Orb
		for _, currentOrb := range ref.Doc.Orbs {
			if position.InRange(currentOrb.NameRange, ref.Params.Position) ||
				position.InRange(currentOrb.Range, ref.Params.Position) {
				orb = currentOrb
			}
		}

		orbInfo, err := ref.Doc.GetOrbInfoFromName(orb.Name, ref.Cache)
		if err == nil && orb.Url.IsLocal {
			return ReferenceHandler{
				Cache:      ref.Cache,
				Params:     ref.Params,
				FoundSteps: ref.FoundSteps,
				Doc:        ref.Doc.FromOrbParsedAttributesToYamlDocument(orbInfo.OrbParsedAttributes),
			}.GetReferences()
		}
	}

	if anchor, found := ref.Doc.GetYamlAnchorAtPosition(ref.Params.Position); found {
		locations := []protocol.Location{}

		for _, aliasRange := range *anchor.References {
			locations = append(
				locations,
				protocol.Location{
					URI:   ref.Params.TextDocument.URI,
					Range: aliasRange,
				},
			)
		}

		return locations, nil
	}

	switch true {
	// Workflow
	case position.InRange(ref.Doc.WorkflowRange, ref.Params.Position):
		cmdName = ref.searchInWorkflows()

	// Job
	case position.InRange(ref.Doc.JobsRange, ref.Params.Position):
		cmdName = ref.searchInJobs()

	// Command
	case position.InRange(ref.Doc.CommandsRange, ref.Params.Position):
		cmdName = ref.searchInCommands()

	// Orb
	case position.InRange(ref.Doc.OrbsRange, ref.Params.Position):
		cmdName = ref.searchInOrbs()
		isOrb = true

	// Executor
	case position.InRange(ref.Doc.ExecutorsRange, ref.Params.Position):
		loc, executorName := ref.getExecutorReferences()
		if len(loc) > 0 {
			return loc, nil
		}
		cmdName = executorName

	// Pipeline parameters
	case position.InRange(ref.Doc.PipelineParametersRange, ref.Params.Position):
		paramName := paramref.NameDefinedAtPos(ref.Doc.PipelineParameters, ref.Params.Position)
		return ref.getReferencesOfParamInRange(paramName, ref.Doc.NodeToRange(ref.Doc.RootNode))
	}

	if paramRefs, err := ref.getParamReferences(cmdName); err == nil {
		return paramRefs, nil
	}

	ref.getStepsOfWorkflows()
	ref.getStepsOfJobs()
	ref.getStepsOfCommands()

	return ref.getReferenceFromSteps(cmdName, isOrb)
}

type StepRangeAndName struct {
	protocol.Range
	Name string
}

func (ref ReferenceHandler) getStepsOfWorkflows() {
	for _, workflow := range ref.Doc.Workflows {
		for _, jobInvocation := range workflow.JobInvocations {
			*ref.FoundSteps = append(*ref.FoundSteps, StepRangeAndName{Name: jobInvocation.JobName, Range: jobInvocation.JobNameRange})
		}
	}
}

func (ref ReferenceHandler) getStepsOfJobs() {
	for _, job := range ref.Doc.Jobs {
		*ref.FoundSteps = append(*ref.FoundSteps, getStepsOfCommandOrJob(job.Steps)...)
	}
}

func (ref ReferenceHandler) getStepsOfCommands() {
	for _, job := range ref.Doc.Commands {
		*ref.FoundSteps = append(*ref.FoundSteps, getStepsOfCommandOrJob(job.Steps)...)
	}
}

func getStepsOfCommandOrJob(steps []ast.Step) []StepRangeAndName {
	res := []StepRangeAndName{}

	for _, step := range steps {
		switch step := step.(type) {
		case ast.NamedStep:
			res = append(res, StepRangeAndName{Name: step.Name, Range: step.Range})

		}
	}

	return res
}

func (ref ReferenceHandler) searchInOrbs() string {
	for _, orb := range ref.Doc.Orbs {
		if position.InRange(orb.NameRange, ref.Params.Position) {
			return orb.Name
		}
	}
	return ""
}

func (ref ReferenceHandler) searchInWorkflows() string {
	for _, workflow := range ref.Doc.Workflows {
		for _, jobInvocation := range workflow.JobInvocations {
			if position.InRange(jobInvocation.JobNameRange, ref.Params.Position) {
				if ref.Doc.DoesCommandOrJobOrExecutorExist(jobInvocation.JobName, false) {
					return jobInvocation.JobName
				}
			}
		}
	}
	return ""
}

func (ref ReferenceHandler) searchInJobs() string {
	for _, job := range ref.Doc.Jobs {
		if position.InRange(job.NameRange, ref.Params.Position) || position.InRange(job.ParametersRange, ref.Params.Position) {
			return job.Name
		}
	}
	return ""
}

func (ref ReferenceHandler) searchInCommands() string {
	for _, command := range ref.Doc.Commands {
		if position.InRange(command.NameRange, ref.Params.Position) || position.InRange(command.ParametersRange, ref.Params.Position) {
			return command.Name
		}
	}
	return ""
}

func (ref ReferenceHandler) getReferenceFromSteps(nameOfStep string, isOrb bool) ([]protocol.Location, error) {
	locations := []protocol.Location{}

	for _, step := range *ref.FoundSteps {
		if step.Name == nameOfStep || (isOrb && strings.HasPrefix(step.Name, nameOfStep+"/")) {
			locations = append(locations, protocol.Location{
				URI:   ref.Params.TextDocument.URI,
				Range: step.Range,
			})
		}
	}

	return locations, nil
}

func (ref ReferenceHandler) getExecutorReferences() ([]protocol.Location, string) {
	executor := ref.Doc.GetExecutorDefinedAtPosition(ref.Params.Position)
	executorName := executor.GetName()

	if position.InRange(executor.GetParametersRange(), ref.Params.Position) {
		return []protocol.Location{}, executorName
	}

	locations := []protocol.Location{}
	for _, job := range ref.Doc.Jobs {
		if job.Executor == executor.GetName() {
			locations = append(locations, protocol.Location{
				URI:   ref.Params.TextDocument.URI,
				Range: job.ExecutorRange,
			})
		}
	}

	return locations, executorName
}

func (ref ReferenceHandler) getParamReferences(cmdName string) ([]protocol.Location, error) {
	var params map[string]ast.Parameter
	var rng protocol.Range

	commandToSearch, ok := ref.Doc.Commands[cmdName]
	if ok && position.InRange(commandToSearch.ParametersRange, ref.Params.Position) {
		params = commandToSearch.Parameters
		rng = commandToSearch.Range
	}

	jobToSearch, ok := ref.Doc.Jobs[cmdName]
	if ok && position.InRange(jobToSearch.ParametersRange, ref.Params.Position) {
		params = jobToSearch.Parameters
		rng = jobToSearch.Range
	}

	executorToSearch, ok := ref.Doc.Executors[cmdName]
	if ok && position.InRange(executorToSearch.GetParametersRange(), ref.Params.Position) {
		params = executorToSearch.GetParameters()
		rng = executorToSearch.GetRange()
	}

	paramName := paramref.NameDefinedAtPos(params, ref.Params.Position)

	if paramName != "" {
		return ref.getReferencesOfParamInRange(paramName, rng)
	}

	return []protocol.Location{}, fmt.Errorf("parameter not found")
}

func (ref ReferenceHandler) getReferencesOfParamInRange(paramName string, rng protocol.Range) ([]protocol.Location, error) {
	content := ref.Doc.Content
	allParamsRef, err := paramref.ReferencesInRange(content, paramName, rng)

	if err != nil {
		return []protocol.Location{}, err
	}

	locations := []protocol.Location{}
	for _, paramRef := range allParamsRef {
		locations = append(locations, protocol.Location{
			URI: ref.Params.TextDocument.URI,
			Range: protocol.Range{
				Start: position.FromIndex(paramRef[0], content),
				End:   position.FromIndex(paramRef[1], content),
			},
		})
	}

	return locations, nil
}
