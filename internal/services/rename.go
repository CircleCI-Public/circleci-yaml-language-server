package languageservice

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	ast2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

// PrepareRename is the name at a position that Rename can rename, or nil when
// there is none there.
func PrepareRename(
	ctx context.Context, params protocol.PrepareRenameParams, cache *cache.Cache, context *session.Settings,
) (*protocol.PrepareRenamePlaceholder, error) {
	doc, err := yamlparser.ParseFromUriWithCache(ctx, params.TextDocument.URI, cache, context)
	if err != nil {
		return nil, err
	}
	defer doc.Close()

	target, ok := renameTargetAt(doc, params.Position)
	if !ok {
		return nil, nil
	}
	if _, err := renameSites(doc, target); err != nil {
		return nil, err
	}
	return &protocol.PrepareRenamePlaceholder{Range: target.at, Placeholder: target.name}, nil
}

// Rename renames the job, command or executor named at a position, where it
// is defined and everywhere it is used.
func Rename(
	ctx context.Context, params protocol.RenameParams, cache *cache.Cache, context *session.Settings,
) (*protocol.WorkspaceEdit, error) {
	doc, err := yamlparser.ParseFromUriWithCache(ctx, params.TextDocument.URI, cache, context)
	if err != nil {
		return nil, err
	}
	defer doc.Close()

	target, ok := renameTargetAt(doc, params.Position)
	if !ok {
		return nil, renameError("there is no job, command or executor here to rename")
	}
	if !validName.MatchString(params.NewName) {
		return nil, jsonrpc2.NewError(jsonrpc2.InvalidParams,
			fmt.Sprintf("%q is not a valid name: use letters, digits, - and _, starting with a letter", params.NewName))
	}
	if target.taken(doc, params.NewName) {
		return nil, renameError(fmt.Sprintf("there is already a %s named %q", target.kind, params.NewName))
	}

	sites, err := renameSites(doc, target)
	if err != nil {
		return nil, err
	}
	edits := make([]protocol.TextEdit, 0, len(sites))
	for _, site := range sites {
		edits = append(edits, protocol.TextEdit{Range: site, NewText: params.NewName})
	}
	return &protocol.WorkspaceEdit{Changes: map[uri.URI][]protocol.TextEdit{params.TextDocument.URI: edits}}, nil
}

// validName is what a new name has to look like. It is stricter than what
// the compiler allows, so that the name never needs quoting, and can't be
// read as an orb's.
var validName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

// codeRequestFailed is the LSP's code for a request that is valid but can't be
// done.
const codeRequestFailed jsonrpc2.Code = -32803

func renameError(message string) error {
	return jsonrpc2.NewError(codeRequestFailed, message)
}

type renameKind string

const (
	renameJob      renameKind = "job"
	renameCommand  renameKind = "command"
	renameExecutor renameKind = "executor"
)

type renameTarget struct {
	kind renameKind
	name string
	// at is the name where the rename was asked for, without any quotes.
	at protocol.Range
}

// taken reports whether name is already one the target's kind uses.
func (target renameTarget) taken(doc yamlparser.YamlDocument, name string) bool {
	switch target.kind {
	case renameJob:
		_, job := doc.Jobs[name]
		_, group := doc.JobGroups[name]
		_, alias := doc.Aliases.Jobs[name]
		return job || group || alias
	case renameCommand:
		_, command := doc.Commands[name]
		_, alias := doc.Aliases.Commands[name]
		return command || alias || doc.IsBuiltIn(name)
	default:
		_, executor := doc.Executors[name]
		_, alias := doc.Aliases.Executors[name]
		return executor || alias
	}
}

// renameTargetAt is the job, command or executor named at pos, whether where
// it is defined or where it is used.
func renameTargetAt(doc yamlparser.YamlDocument, pos protocol.Position) (renameTarget, bool) {
	content := doc.Content
	found := func(kind renameKind, name string, rng protocol.Range) (renameTarget, bool) {
		at, ok := nameWithin(content, rng, name)
		if !ok || !position.InRange(at, pos) {
			return renameTarget{}, false
		}
		return renameTarget{kind: kind, name: name, at: at}, true
	}

	for name, job := range doc.Jobs {
		if target, ok := found(renameJob, name, job.NameRange); ok {
			return target, true
		}
		if _, ok := doc.Executors[job.Executor]; ok {
			if target, ok := found(renameExecutor, job.Executor, job.ExecutorRange); ok {
				return target, true
			}
		}
	}
	for name, command := range doc.Commands {
		if target, ok := found(renameCommand, name, command.NameRange); ok {
			return target, true
		}
	}
	for name, executor := range doc.Executors {
		if target, ok := found(renameExecutor, name, executor.GetNameRange()); ok {
			return target, true
		}
	}

	for _, invocations := range invocationLists(doc) {
		for _, invocation := range invocations {
			if _, ok := doc.Jobs[invocation.JobName]; ok {
				if target, ok := found(renameJob, invocation.JobName, invocation.JobNameRange); ok {
					return target, true
				}
			}
			for _, require := range invocation.Requires {
				job, ok := requiredJob(doc, invocations, require.Name)
				if !ok {
					continue
				}
				if at, ok := jobNameInRequire(content, require, job); ok && position.InRange(at, pos) {
					return renameTarget{kind: renameJob, name: job, at: at}, true
				}
			}
		}
	}

	for _, step := range allNamedSteps(doc) {
		if _, ok := doc.Commands[step.Name]; ok {
			if target, ok := found(renameCommand, step.Name, step.Range); ok {
				return target, true
			}
		}
	}

	return renameTarget{}, false
}

// renameSites are the ranges of every mention of target's name that renaming
// it changes. It fails rather than leave a mention out.
func renameSites(doc yamlparser.YamlDocument, target renameTarget) ([]protocol.Range, error) {
	aliases := map[renameKind]map[string]ast2.Alias{
		renameJob:      doc.Aliases.Jobs,
		renameCommand:  doc.Aliases.Commands,
		renameExecutor: doc.Aliases.Executors,
	}
	if _, ok := aliases[target.kind][target.name]; ok {
		return nil, renameError(fmt.Sprintf("%s %q is defined through a YAML alias, which can't be renamed",
			target.kind, target.name))
	}

	var mentions []protocol.Range
	var sites []protocol.Range
	switch target.kind {
	case renameJob:
		mentions = append(mentions, doc.Jobs[target.name].NameRange)
		for _, invocations := range invocationLists(doc) {
			for _, invocation := range invocations {
				if invocation.JobName == target.name {
					mentions = append(mentions, invocation.JobNameRange)
				}
				for _, require := range invocation.Requires {
					if job, ok := requiredJob(doc, invocations, require.Name); !ok || job != target.name {
						continue
					}
					site, ok := jobNameInRequire(doc.Content, require, target.name)
					if !ok {
						return nil, cantRename(target, require.Range)
					}
					sites = append(sites, site)
				}
			}
		}

	case renameCommand:
		mentions = append(mentions, doc.Commands[target.name].NameRange)
		for _, step := range allNamedSteps(doc) {
			if step.Name == target.name {
				mentions = append(mentions, step.Range)
			}
		}

	case renameExecutor:
		defined := doc.Executors[target.name].GetNameRange()
		mentions = append(mentions, defined)
		at := protocol.TextDocumentPositionParams{Position: defined.Start}
		ref := ReferenceHandler{Doc: doc, Params: protocol.ReferenceParams{TextDocumentPositionParams: at}}
		locations, _ := ref.getExecutorReferences()
		for _, location := range locations {
			mentions = append(mentions, location.Range)
		}
	}

	for _, mention := range mentions {
		site, ok := nameWithin(doc.Content, mention, target.name)
		if !ok {
			return nil, cantRename(target, mention)
		}
		sites = append(sites, site)
	}

	slices.SortFunc(sites, func(a, b protocol.Range) int { return position.Compare(a.Start, b.Start) })
	return slices.CompactFunc(sites, position.AreRangeEqual), nil
}

func cantRename(target renameTarget, at protocol.Range) error {
	return renameError(fmt.Sprintf("can't rename %s %q: its mention on line %d isn't one this can change",
		target.kind, target.name, at.Start.Line+1))
}

// invocationLists are the lists of job invocations that a requires can name
// a job among: each workflow's, and each job group's.
func invocationLists(doc yamlparser.YamlDocument) [][]ast2.JobInvocation {
	var lists [][]ast2.JobInvocation
	for _, workflow := range doc.Workflows {
		lists = append(lists, workflow.JobInvocations)
	}
	for _, group := range doc.JobGroups {
		lists = append(lists, group.JobInvocations)
	}
	return lists
}

// requiredJob is the job a requires among invocations names by the job's own
// name, rather than by a name an invocation gives it.
func requiredJob(doc yamlparser.YamlDocument, invocations []ast2.JobInvocation, required string) (string, bool) {
	for _, invocation := range invocations {
		if _, ok := doc.Jobs[invocation.JobName]; !ok || !namedByJob(invocation) {
			continue
		}
		if required == invocation.JobName && invocation.StepName == invocation.JobName {
			return invocation.JobName, true
		}
		if _, named := invocation.MatrixParams["name"]; !named && slices.Contains(invocation.MatrixNames, required) &&
			strings.HasPrefix(required, invocation.JobName+"-") {
			return invocation.JobName, true
		}
	}
	return "", false
}

// namedByJob reports whether an invocation takes its name from its job's,
// rather than from a `name` of its own.
func namedByJob(invocation ast2.JobInvocation) bool {
	return position.AreRangeEqual(invocation.StepNameRange, invocation.JobNameRange)
}

// jobNameInRequire is where job's name is in a requires naming it, which is
// all of it, or the start of the name of one of a matrix's jobs.
func jobNameInRequire(content []byte, require ast2.Require, job string) (protocol.Range, bool) {
	if require.Name == job {
		return nameWithin(content, require.Range, job)
	}
	whole, ok := nameWithin(content, require.Range, require.Name)
	if !ok {
		return protocol.Range{}, false
	}
	end := whole.Start
	end.Character += uint32(len(job))
	return protocol.Range{Start: whole.Start, End: end}, true
}

// allNamedSteps are the steps that name a command, wherever a list of steps
// can be: in jobs and commands, before and after a job in a workflow, and as
// the value of a steps parameter.
func allNamedSteps(doc yamlparser.YamlDocument) []ast2.NamedStep {
	var named []ast2.NamedStep
	var addSteps func(steps []ast2.Step)
	var addValue func(value ast2.ParameterValue, isSteps bool)
	addValue = func(value ast2.ParameterValue, isSteps bool) {
		switch v := value.Value.(type) {
		case []ast2.Step:
			addSteps(v)
		case []ast2.ParameterValue:
			for _, item := range v {
				addValue(item, isSteps)
			}
		case string:
			// An argument isn't parsed knowing its parameter's type, so a
			// step in one that is only a name is read as a string.
			if isSteps {
				named = append(named, ast2.NamedStep{Name: strings.Trim(v, `"'`), Range: value.ValueRange})
			}
		}
	}
	addArguments := func(arguments map[string]ast2.ParameterValue, declared map[string]ast2.Parameter) {
		for name, argument := range arguments {
			_, isSteps := declared[name].(ast2.StepsParameter)
			addValue(argument, isSteps)
		}
	}
	addSteps = func(steps []ast2.Step) {
		for _, step := range steps {
			switch step := step.(type) {
			case ast2.NamedStep:
				named = append(named, step)
				addArguments(step.Parameters, doc.Commands[step.Name].Parameters)
			case ast2.Steps:
				addSteps(step.Steps)
			}
		}
	}
	addDefaults := func(parameters map[string]ast2.Parameter) {
		for _, parameter := range parameters {
			if steps, ok := parameter.(ast2.StepsParameter); ok {
				addValue(steps.Default, true)
			}
		}
	}

	for _, job := range doc.Jobs {
		addSteps(job.Steps)
		addDefaults(job.Parameters)
	}
	for _, command := range doc.Commands {
		addSteps(command.Steps)
		addDefaults(command.Parameters)
	}
	for _, invocations := range invocationLists(doc) {
		for _, invocation := range invocations {
			addSteps(invocation.PreSteps)
			addSteps(invocation.PostSteps)
			addArguments(invocation.Parameters, doc.Jobs[invocation.JobName].Parameters)
		}
	}
	return named
}

// nameWithin is where name is in rng, as a whole word: the one place it is,
// leaving out any quotes around it. A range that holds the name more than
// once, such as `name: name`, can't say which is meant.
func nameWithin(content []byte, rng protocol.Range, name string) (protocol.Range, bool) {
	start := position.ToIndex(rng.Start, content)
	end := position.ToIndex(rng.End, content)
	if name == "" || start < 0 || end > len(content) || start >= end {
		return protocol.Range{}, false
	}
	text := string(content[start:end])

	found := -1
	for offset := 0; ; {
		i := strings.Index(text[offset:], name)
		if i < 0 {
			break
		}
		i += offset
		offset = i + 1
		after := i + len(name)
		if (i > 0 && isNameByte(text[i-1])) || (after < len(text) && isNameByte(text[after])) {
			continue
		}
		if found >= 0 {
			return protocol.Range{}, false
		}
		found = i
	}
	if found < 0 {
		return protocol.Range{}, false
	}

	from := position.FromIndex(start+found, content)
	return protocol.Range{Start: from, End: position.Advance(from, []byte(name))}, true
}

// isNameByte reports whether b can be part of a job's, command's or
// executor's name, or of the name of an orb's.
func isNameByte(b byte) bool {
	return b == '-' || b == '_' || b == '/' || b == '.' || b == '@' ||
		('a' <= b && b <= 'z') || ('A' <= b && b <= 'Z') || ('0' <= b && b <= '9')
}
