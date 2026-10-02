package hover

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

// Step is the hover for the name of a step that runs a command, whether the
// config's own, an inline orb's or an orb's, directly or through an alias:
// the command's description and parameters.
func Step(ctx context.Context, doc yamlparser.YamlDocument, c *cache.Cache, pos protocol.Position) (string, bool) {
	if step, ok := namedStepAt(pos, doc.Jobs, doc.Commands); ok {
		return describeCommand(ctx, doc, c, step.Name)
	}

	// An inline orb's steps name its commands without the orb's prefix.
	for _, orb := range doc.LocalOrbInfo {
		if step, ok := namedStepAt(pos, orb.Jobs, orb.Commands); ok {
			return describeCommand(ctx, doc.FromOrbParsedAttributesToYamlDocument(orb.OrbParsedAttributes), c, step.Name)
		}
	}
	return "", false
}

func describeCommand(ctx context.Context, doc yamlparser.YamlDocument, c *cache.Cache, name string) (string, bool) {
	command, ok := doc.ResolveCommand(ctx, name, c)
	if !ok {
		return "", false
	}
	return describe(name, "command", command.Description, command.Parameters), true
}

// namedStepAt is the step, in one of the jobs or commands, whose name the
// cursor is on.
func namedStepAt(pos protocol.Position, jobs map[string]ast.Job, commands map[string]ast.Command) (ast.NamedStep, bool) {
	var lists [][]ast.Step
	for _, job := range jobs {
		lists = append(lists, job.Steps)
	}
	for _, command := range commands {
		lists = append(lists, command.Steps)
	}

	for _, steps := range lists {
		for _, step := range steps {
			if named, ok := step.(ast.NamedStep); ok && named.Name != "" && position.InRange(named.Range, pos) {
				return named, true
			}
		}
	}
	return ast.NamedStep{}, false
}

// describe is the hover for a definition: its name and kind, its description
// and its parameters.
func describe(name, kind, description string, params map[string]ast.Parameter) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**%s** %s", name, kind)
	if description != "" {
		fmt.Fprintf(&b, "\n\n%s", description)
	}
	writeParameters(&b, params)
	return b.String()
}

// writeParameters lists the parameters by name, each with its type, its
// default or that it's required, and its description.
func writeParameters(b *strings.Builder, params map[string]ast.Parameter) {
	inputs := make([]input, 0, len(params))
	for name, param := range params {
		value, hasDefault := defaultOf(param)
		inputs = append(inputs, input{
			name:        name,
			kind:        param.GetType(),
			value:       value,
			hasDefault:  hasDefault,
			required:    !param.IsOptional(),
			description: param.GetDescription(),
		})
	}
	writeInputs(b, "Parameters", inputs)
}

// input is a parameter, or a function's flag, as a hover lists it.
type input struct {
	name, kind, value, description string
	hasDefault, required           bool
}

// writeInputs lists the inputs under a heading, sorted by name.
func writeInputs(b *strings.Builder, heading string, inputs []input) {
	if len(inputs) == 0 {
		return
	}
	slices.SortFunc(inputs, func(a, b input) int { return strings.Compare(a.name, b.name) })

	fmt.Fprintf(b, "\n\n%s:\n", heading)
	for _, in := range inputs {
		fmt.Fprintf(b, "\n- `%s` (%s", in.name, in.kind)
		if in.hasDefault {
			fmt.Fprintf(b, ", default `%s`", in.value)
		} else if in.required {
			b.WriteString(", required")
		}
		b.WriteString(")")
		if in.description != "" {
			// Indented, so that a description of several lines stays in its
			// list item.
			fmt.Fprintf(b, ": %s", strings.ReplaceAll(in.description, "\n", "\n  "))
		}
	}
}

// defaultOf is a parameter's default written as a value, for the types whose
// default is a scalar.
func defaultOf(param ast.Parameter) (string, bool) {
	if !param.IsOptional() {
		return "", false
	}

	switch p := param.(type) {
	case ast.StringParameter:
		return p.Default, true
	case ast.BooleanParameter:
		return fmt.Sprint(p.Default), true
	case ast.IntegerParameter:
		return fmt.Sprint(p.Default), true
	case ast.EnumParameter:
		return p.Default, true
	case ast.ExecutorParameter:
		return p.Default, true
	case ast.EnvVariableParameter:
		return p.Default, true
	}
	return "", false
}
