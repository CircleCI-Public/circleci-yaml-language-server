package languageservice

import (
	"context"
	"fmt"
	"strings"

	"go.lsp.dev/protocol"

	ast2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/paramref"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

func References(ctx context.Context, params protocol.ReferenceParams, cache *cache.Cache, context *session.Settings) ([]protocol.Location, error) {
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

	return ref.GetReferences(ctx)
}

type ReferenceHandler struct {
	Doc        yamlparser.YamlDocument
	Params     protocol.ReferenceParams
	Cache      *cache.Cache
	FoundSteps *[]StepRangeAndName
}

func (ref ReferenceHandler) GetReferences(ctx context.Context) ([]protocol.Location, error) {
	cmdName := ""
	isOrb := false

	if position.InRange(ref.Doc.OrbsRange, ref.Params.Position) {
		var orb ast2.Orb
		for _, currentOrb := range ref.Doc.Orbs {
			if position.InRange(currentOrb.NameRange, ref.Params.Position) ||
				position.InRange(currentOrb.Range, ref.Params.Position) {
				orb = currentOrb
			}
		}

		orbInfo, err := ref.Doc.GetOrbInfoFromName(ctx, orb.Name, ref.Cache)
		if err == nil && orb.Url.IsLocal {
			return ReferenceHandler{
				Cache:      ref.Cache,
				Params:     ref.Params,
				FoundSteps: ref.FoundSteps,
				Doc:        ref.Doc.FromOrbParsedAttributesToYamlDocument(orbInfo.OrbParsedAttributes),
			}.GetReferences(ctx)
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

func getStepsOfCommandOrJob(steps []ast2.Step) []StepRangeAndName {
	res := []StepRangeAndName{}

	for _, step := range steps {
		switch step := step.(type) {
		case ast2.NamedStep:
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
	return ref.aliasNamedAt(ref.Doc.Aliases.Jobs)
}

func (ref ReferenceHandler) searchInCommands() string {
	for _, command := range ref.Doc.Commands {
		if position.InRange(command.NameRange, ref.Params.Position) || position.InRange(command.ParametersRange, ref.Params.Position) {
			return command.Name
		}
	}
	return ref.aliasNamedAt(ref.Doc.Aliases.Commands)
}

func (ref ReferenceHandler) aliasNamedAt(aliases map[string]ast2.Alias) string {
	for _, alias := range aliases {
		if position.InRange(alias.NameRange, ref.Params.Position) {
			return alias.Name
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
	if alias := ref.aliasNamedAt(ref.Doc.Aliases.Executors); alias != "" {
		executorName = alias
	}

	if position.InRange(executor.GetParametersRange(), ref.Params.Position) {
		return []protocol.Location{}, executorName
	}

	locations := []protocol.Location{}
	add := func(rng protocol.Range) {
		locations = append(locations, protocol.Location{URI: ref.Params.TextDocument.URI, Range: rng})
	}

	for _, job := range ref.Doc.Jobs {
		if job.Executor == executorName {
			add(job.ExecutorRange)
			continue
		}

		// A job with `executor: << parameters.x >>` uses whichever executor x
		// names: by default, or as the argument or matrix value of a job that
		// invokes it.
		if !paramref.IsOnlyParameter(job.Executor) {
			continue
		}
		path, param := paramref.ExtractName(job.Executor)
		if path != "parameters."+param {
			continue
		}

		if definition, ok := job.Parameters[param]; ok && parameterDefault(definition) == executorName {
			add(definition.GetDefaultRange())
		}
		for _, rng := range ref.executorArguments(job.Name, param, executorName) {
			add(rng)
		}
	}

	return locations, executorName
}

// executorArguments are where the invocations of job give its parameter param
// the value executorName, directly or among a matrix's values.
func (ref ReferenceHandler) executorArguments(job, param, executorName string) []protocol.Range {
	var invocations []ast2.JobInvocation
	for _, workflow := range ref.Doc.Workflows {
		invocations = append(invocations, workflow.JobInvocations...)
	}
	for _, group := range ref.Doc.JobGroups {
		invocations = append(invocations, group.JobInvocations...)
	}

	var ranges []protocol.Range
	for _, invocation := range invocations {
		if invocation.JobName != job {
			continue
		}

		if argument, ok := invocation.Parameters[param]; ok && namesValue(argument, executorName) {
			ranges = append(ranges, argument.Range)
		}
		for _, values := range invocation.MatrixParams[param] {
			list, _ := values.Value.([]ast2.ParameterValue)
			for _, value := range list {
				if namesValue(value, executorName) {
					ranges = append(ranges, value.ValueRange)
				}
			}
		}
	}

	return ranges
}

// namesValue reports whether a parameter's value is the string name, quoted
// or not.
func namesValue(value ast2.ParameterValue, name string) bool {
	text, ok := value.Value.(string)
	return ok && strings.Trim(text, `"'`) == name
}

// parameterDefault is the default of a parameter that can name an executor.
func parameterDefault(parameter ast2.Parameter) string {
	var text string
	switch parameter := parameter.(type) {
	case ast2.ExecutorParameter:
		text = parameter.Default
	case ast2.StringParameter:
		text = parameter.Default
	case ast2.EnumParameter:
		text = parameter.Default
	}
	return strings.Trim(text, `"'`)
}

func (ref ReferenceHandler) getParamReferences(cmdName string) ([]protocol.Location, error) {
	var params map[string]ast2.Parameter
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
