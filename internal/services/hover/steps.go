package hover

import (
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
// config's own, an inline orb's or an orb's: the command's description and
// parameters.
func Step(doc yamlparser.YamlDocument, c *cache.Cache, pos protocol.Position) (string, bool) {
	if step, ok := namedStepAt(pos, doc.Jobs, doc.Commands); ok {
		if command, ok := doc.Commands[step.Name]; ok {
			return describe(step.Name, "command", command.Description, command.Parameters), true
		}
		if orb, name, ok := orbOf(doc, c, step.Name); ok {
			if command, ok := orb.Commands[name]; ok {
				return describe(step.Name, "command", command.Description, command.Parameters), true
			}
		}
		return "", false
	}

	// An inline orb's steps name its commands without the orb's prefix.
	for _, orb := range doc.LocalOrbInfo {
		if step, ok := namedStepAt(pos, orb.Jobs, orb.Commands); ok {
			if command, ok := orb.Commands[step.Name]; ok {
				return describe(step.Name, "command", command.Description, command.Parameters), true
			}
			return "", false
		}
	}
	return "", false
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

// orbOf is the orb a reference such as `orb/name` points into, when the
// config declares that orb, and the name within it.
func orbOf(doc yamlparser.YamlDocument, c *cache.Cache, reference string) (*ast.OrbInfo, string, bool) {
	orbName, name, ok := strings.Cut(reference, "/")
	if !ok {
		return nil, "", false
	}
	if _, declared := doc.Orbs[orbName]; !declared {
		return nil, "", false
	}
	orb, err := doc.GetOrbInfoFromName(orbName, c)
	if err != nil || orb == nil {
		return nil, "", false
	}
	return orb, name, true
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
	if len(params) == 0 {
		return
	}

	names := make([]string, 0, len(params))
	for name := range params {
		names = append(names, name)
	}
	slices.Sort(names)

	b.WriteString("\n\nParameters:\n")
	for _, name := range names {
		param := params[name]
		fmt.Fprintf(b, "\n- `%s` (%s", name, param.GetType())
		if value, ok := defaultOf(param); ok {
			fmt.Fprintf(b, ", default `%s`", value)
		} else if !param.IsOptional() {
			b.WriteString(", required")
		}
		b.WriteString(")")
		if description := param.GetDescription(); description != "" {
			fmt.Fprintf(b, ": %s", description)
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
