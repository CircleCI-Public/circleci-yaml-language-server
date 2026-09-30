package validate

import (
	"bytes"
	"slices"
	"strings"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/paramref"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/pipelinevalues"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/yamltree"
)

var textualTypes = []string{"string", "enum", "env_var_name"}

// referencedType returns the type of the parameter or pipeline value that
// text is nothing but a reference to. The compiler keeps that type, so such
// a value is only a string when what it refers to is. parameters are the ones
// in scope for `<< parameters.x >>`.
func (val Validate) referencedType(text string, parameters map[string]ast.Parameter) (string, bool) {
	text = strings.TrimSpace(text)
	if name, ok := paramref.OnlyPipelineValue(text); ok {
		value, known := pipelinevalues.Lookup(name)
		return parameterTypeOf(value), known
	}
	if !paramref.IsOnlyParameter(text) {
		return "", false
	}
	full, name := paramref.ExtractName(text)
	if strings.HasPrefix(full, "pipeline.") {
		parameters = val.Doc.PipelineParameters
	}
	param, ok := parameters[name]
	if !ok {
		return "", false
	}
	return param.GetType(), true
}

// validateExecutorNameReference reports a job's executor that refers to a
// parameter or pipeline value that can't be an executor's name.
func (val Validate) validateExecutorNameReference(job ast.Job) {
	paramType, ok := val.referencedType(job.Executor, job.Parameters)
	if !ok || paramType == "executor" || slices.Contains(textualTypes, paramType) {
		return
	}
	rng := job.ExecutorRange
	start := position.ToIndex(rng.Start, val.Doc.Content)
	if at := bytes.Index(val.Doc.Content[start:], []byte(job.Executor)); at >= 0 {
		rng = protocol.Range{
			Start: position.FromIndex(start+at, val.Doc.Content),
			End:   position.FromIndex(start+at+len(job.Executor), val.Doc.Content),
		}
	}
	val.addDiagnostic(diagnostic.Error(rng,
		"Executor invocation "+job.Executor+" must resolve to an executor name"))
}

// ValidateMatchesValues reports a logic statement's `matches` value that
// refers to a parameter or pipeline value that isn't a string.
func (val Validate) ValidateMatchesValues() {
	for node := range yamltree.Walk(val.Doc.RootNode) {
		if !isPair(node) || val.Doc.IsUnderUnreadTopLevelKey(node) ||
			val.pairKey(node) != "value" || val.pairKey(enclosingPair(node)) != "matches" {
			continue
		}
		_, value := val.Doc.GetKeyValueNodes(node)
		if value == nil {
			continue
		}
		parameters := val.Doc.GetParamsWithPosition(val.Doc.NodeToRange(value).Start)
		paramType, ok := val.referencedType(val.Doc.ScalarText(value), parameters)
		if ok && !slices.Contains(textualTypes, paramType) {
			val.addDiagnostic(diagnostic.ErrorFromNode(value,
				"matches: value must produce a string, but the referenced parameter is not of type string"))
		}
	}
}

// validateExecutorArgumentReference reports an executor argument given as a
// map whose name refers to something that can't be an executor's name. The
// compiler reads that map inside the invoked job, so its parameters are the
// job's.
func (val Validate) validateExecutorArgumentReference(arg ast.ParameterValue, jobParameters map[string]ast.Parameter) {
	fields, ok := arg.Value.(map[string]ast.ParameterValue)
	if !ok {
		return
	}
	name, ok := fields["name"].Value.(string)
	if !ok {
		return
	}
	paramType, ok := val.referencedType(name, jobParameters)
	if !ok || paramType == "executor" || slices.Contains(textualTypes, paramType) {
		return
	}
	val.addDiagnostic(diagnostic.Error(fields["name"].ValueRange,
		"Executor invocation "+strings.TrimSpace(name)+" must resolve to an executor name"))
}

// executorArgumentParametersAt returns the parameters of the job invoked
// around pos when pos is in an executor argument given as a map, which the
// compiler reads inside that job.
func (val Validate) executorArgumentParametersAt(pos protocol.Position) (map[string]ast.Parameter, bool) {
	for invocation := range invocations(&val.Doc) {
		if !position.InRange(invocation.JobInvocationRange, pos) {
			continue
		}
		for _, arg := range invocation.Parameters {
			if arg.Type != "map" || !position.InRange(arg.Range, pos) {
				continue
			}
			jobParameters := val.Doc.GetDefinedParams(invocation.JobName, parser.JobEntity, val.Cache)
			if param, ok := jobParameters[arg.Name]; ok && param.GetType() == "executor" {
				return jobParameters, true
			}
		}
	}
	return nil, false
}
