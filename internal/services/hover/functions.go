package hover

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

// FunctionStep is the hover for the name of a step that runs a declared
// function, or one of its commands: the description and flags its descriptor
// in the functions catalog gives.
func FunctionStep(ctx context.Context, doc yamlparser.YamlDocument, c *cache.Cache, pos protocol.Position) (string, bool) {
	step, ok := namedStepAt(pos, doc.Jobs, doc.Commands)
	if !ok {
		return "", false
	}
	function, command, ok := doc.FunctionForStep(step.Name)
	if !ok {
		return "", false
	}

	_, descriptor, err := doc.LookUpFunction(ctx, function, c)
	if err != nil || descriptor == nil {
		return "", false
	}
	if command == "" {
		return describeFunction(step.Name, "function", descriptor.Description, descriptor.Flags), true
	}
	if found, ok := descriptor.Commands[command]; ok {
		return describeFunction(step.Name, "function command", found.Description, found.Flags), true
	}
	return "", false
}

// FunctionDeclaration is the hover for a declaration under `functions:`: the
// description, flags and example config of the version it declares.
func FunctionDeclaration(ctx context.Context, doc yamlparser.YamlDocument, c *cache.Cache, pos protocol.Position) (string, bool) {
	function, ok := declarationAt(doc, pos)
	if !ok {
		return "", false
	}

	_, descriptor, err := doc.LookUpFunction(ctx, function, c)
	if err != nil || descriptor == nil {
		return "", false
	}
	text := describeFunction(function.Alias, "function", descriptor.Description, descriptor.Flags)
	if descriptor.Example != "" {
		text += "\n\nExample:\n\n```yaml\n" + strings.TrimRight(descriptor.Example, "\n") + "\n```"
	}
	return text, true
}

// FunctionFlag is the hover for a flag a function step passes under `with`:
// its type, default and description from the function's descriptor.
func FunctionFlag(ctx context.Context, doc yamlparser.YamlDocument, c *cache.Cache, pos protocol.Position) (string, bool) {
	step, name, ok := functionArgumentAt(pos, doc.Jobs, doc.Commands)
	if !ok {
		return "", false
	}
	function, command, ok := doc.FunctionForStep(step.Name)
	if !ok {
		return "", false
	}

	_, descriptor, err := doc.LookUpFunction(ctx, function, c)
	if err != nil || descriptor == nil {
		return "", false
	}
	flags := descriptor.Flags
	if command != "" {
		flags = descriptor.Commands[command].Flags
	}
	index := slices.IndexFunc(flags, func(flag circleci.FunctionFlag) bool { return flag.Name == name })
	if index == -1 {
		return "", false
	}
	return describeFlag(step.Name, flags[index]), true
}

// functionArgumentAt is the step, and the name of the argument under its
// `with`, whose key the cursor is on. The value is left to other hovers, such
// as a parameter reference's.
func functionArgumentAt(pos protocol.Position, jobs map[string]ast.Job, commands map[string]ast.Command) (ast.NamedStep, string, bool) {
	for _, steps := range stepLists(jobs, commands) {
		for _, step := range steps {
			named, ok := step.(ast.NamedStep)
			if !ok {
				continue
			}
			with, ok := named.Parameters["with"]
			if !ok || with.Type != "map" {
				continue
			}
			arguments, _ := with.Value.(map[string]ast.ParameterValue)
			for name, argument := range arguments {
				key := protocol.Range{Start: argument.Range.Start, End: argument.Range.Start}
				key.End.Character += uint32(len(name))
				if position.InRange(key, pos) {
					return named, name, true
				}
			}
		}
	}
	return ast.NamedStep{}, "", false
}

func describeFlag(step string, flag circleci.FunctionFlag) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**%s** flag of **%s**\n\n(%s", flag.Name, step, flag.Type)
	if value, ok := flag.DefaultValue(); ok {
		fmt.Fprintf(&b, ", default `%s`", value)
	}
	b.WriteString(")")
	if flag.Description != "" {
		fmt.Fprintf(&b, ": %s", flag.Description)
	}
	return b.String()
}

func declarationAt(doc yamlparser.YamlDocument, pos protocol.Position) (ast.Function, bool) {
	for _, function := range doc.Functions {
		if position.InRange(function.Range, pos) {
			return function, true
		}
	}
	return ast.Function{}, false
}

func describeFunction(name, kind, description string, flags []circleci.FunctionFlag) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**%s** %s", name, kind)
	if description != "" {
		fmt.Fprintf(&b, "\n\n%s", description)
	}

	inputs := make([]input, 0, len(flags))
	for _, flag := range flags {
		inputs = append(inputs, input{
			name:        flag.Name,
			kind:        flag.Type,
			value:       fmt.Sprint(flag.Default),
			hasDefault:  flag.Default != nil,
			description: flag.Description,
		})
	}
	writeInputs(&b, "Flags, under `with`", inputs)
	return b.String()
}
