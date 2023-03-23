package languageservice

import (
	"fmt"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	utils "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func References(params protocol.ReferenceParams, cache *utils.Cache, context *utils.LsContext) ([]protocol.Location, error) {
	yamlDocument, err := yamlparser.ParseFromUriWithCache(params.TextDocument.URI, cache, context)

	if err != nil {
		return nil, err
	}

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
	Cache      *utils.Cache
	FoundSteps *[]StepRangeAndName
}

func (ref ReferenceHandler) GetReferences() ([]protocol.Location, error) {
	cmdName := ""
	isOrb := false

	if utils.PosInRange(ref.Doc.OrbsRange, ref.Params.Position) {
		var orb ast.Orb
		for _, currentOrb := range ref.Doc.Orbs {
			if utils.PosInRange(currentOrb.NameRange, ref.Params.Position) ||
				utils.PosInRange(currentOrb.Range, ref.Params.Position) {
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
	case utils.PosInRange(ref.Doc.WorkflowRange, ref.Params.Position):
		cmdName = ref.searchInWorkflows()

	// Job
	case utils.PosInRange(ref.Doc.JobsRange, ref.Params.Position):
		cmdName = ref.searchInJobs()

	// Command
	case utils.PosInRange(ref.Doc.CommandsRange, ref.Params.Position):
		cmdName = ref.searchInCommands()

	// Orb
	case utils.PosInRange(ref.Doc.OrbsRange, ref.Params.Position):
		cmdName = ref.searchInOrbs()
		isOrb = true

	// Executor
	case utils.PosInRange(ref.Doc.ExecutorsRange, ref.Params.Position):
		loc, executorName := ref.getExecutorReferences()
		if len(loc) > 0 {
			return loc, nil
		}
		cmdName = executorName

	// Pipeline parameters
	case utils.PosInRange(ref.Doc.PipelineParametersRange, ref.Params.Position):
		paramName := utils.GetParamNameDefinedAtPos(ref.Doc.PipelineParameters, ref.Params.Position)
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
		for _, jobRef := range workflow.JobRefs {
			*ref.FoundSteps = append(*ref.FoundSteps, StepRangeAndName{Name: jobRef.JobName, Range: jobRef.JobNameRange})
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
		if utils.PosInRange(orb.NameRange, ref.Params.Position) {
			return orb.Name
		}
	}
	return ""
}

func (ref ReferenceHandler) searchInWorkflows() string {
	for _, workflow := range ref.Doc.Workflows {
		for _, jobRef := range workflow.JobRefs {
			if utils.PosInRange(jobRef.JobNameRange, ref.Params.Position) {
				if ref.Doc.DoesCommandOrJobOrExecutorExist(jobRef.JobName, false) {
					return jobRef.JobName
				}
			}
		}
	}
	return ""
}

func (ref ReferenceHandler) searchInJobs() string {
	for _, job := range ref.Doc.Jobs {
		if utils.PosInRange(job.NameRange, ref.Params.Position) || utils.PosInRange(job.ParametersRange, ref.Params.Position) {
			return job.Name
		}
	}
	return ""
}

func (ref ReferenceHandler) searchInCommands() string {
	for _, command := range ref.Doc.Commands {
		if utils.PosInRange(command.NameRange, ref.Params.Position) || utils.PosInRange(command.ParametersRange, ref.Params.Position) {
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

	if utils.PosInRange(executor.GetParametersRange(), ref.Params.Position) {
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
	if ok && utils.PosInRange(commandToSearch.ParametersRange, ref.Params.Position) {
		params = commandToSearch.Parameters
		rng = commandToSearch.Range
	}

	jobToSearch, ok := ref.Doc.Jobs[cmdName]
	if ok && utils.PosInRange(jobToSearch.ParametersRange, ref.Params.Position) {
		params = jobToSearch.Parameters
		rng = jobToSearch.Range
	}

	executorToSearch, ok := ref.Doc.Executors[cmdName]
	if ok && utils.PosInRange(executorToSearch.GetParametersRange(), ref.Params.Position) {
		params = executorToSearch.GetParameters()
		rng = executorToSearch.GetRange()
	}

	paramName := utils.GetParamNameDefinedAtPos(params, ref.Params.Position)

	if paramName != "" {
		return ref.getReferencesOfParamInRange(paramName, rng)
	}

	return []protocol.Location{}, fmt.Errorf("parameter not found")
}

func (ref ReferenceHandler) getReferencesOfParamInRange(paramName string, rng protocol.Range) ([]protocol.Location, error) {
	content := ref.Doc.Content
	allParamsRef, err := utils.GetReferencesOfParamInRange(content, paramName, rng)

	if err != nil {
		return []protocol.Location{}, err
	}

	locations := []protocol.Location{}
	for _, paramRef := range allParamsRef {
		locations = append(locations, protocol.Location{
			URI: ref.Params.TextDocument.URI,
			Range: protocol.Range{
				Start: utils.IndexToPos(paramRef[0], content),
				End:   utils.IndexToPos(paramRef[1], content),
			},
		})
	}

	return locations, nil
}
