package hover

import (
	"bytes"
	"fmt"
	"strings"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/paramref"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

// ParameterReference is the hover for a reference to a parameter, such as
// `<< parameters.os >>`, `<<# parameters.verbose >>` or
// `<< pipeline.parameters.setup_go >>`: the type, default and description of
// the parameter it names.
func ParameterReference(doc yamlparser.YamlDocument, _ *cache.Cache, pos protocol.Position) (string, bool) {
	lineStart := position.ToIndex(protocol.Position{Line: pos.Line}, doc.Content)
	lineEnd := bytes.IndexByte(doc.Content[lineStart:], '\n')
	if lineEnd == -1 {
		lineEnd = len(doc.Content) - lineStart
	}
	line := doc.Content[lineStart : lineStart+lineEnd]
	column := int(pos.Character)

	for _, match := range paramref.Pattern.FindAllSubmatchIndex(line, -1) {
		if column < match[0] || column > match[1] {
			continue
		}
		name := line[match[4]:match[5]]

		if string(line[match[2]:match[3]]) == "pipeline.parameters" {
			parameter, ok := doc.PipelineParameters[string(name)]
			if !ok {
				return "", false
			}
			return describeParameter(parameter, "A pipeline parameter."), true
		}

		owner, parameters, ok := parametersAround(doc, pos)
		if !ok {
			return "", false
		}
		parameter, ok := parameters[string(name)]
		if !ok {
			return "", false
		}
		return describeParameter(parameter, fmt.Sprintf("A parameter of %s.", owner)), true
	}
	return "", false
}

// parametersAround returns the parameters of the job, command or executor
// pos is in, and which one that is.
func parametersAround(doc yamlparser.YamlDocument, pos protocol.Position) (string, map[string]ast.Parameter, bool) {
	for name, job := range doc.Jobs {
		if position.InRange(job.Range, pos) {
			return fmt.Sprintf("the job `%s`", name), job.Parameters, true
		}
	}
	for name, command := range doc.Commands {
		if position.InRange(command.Range, pos) {
			return fmt.Sprintf("the command `%s`", name), command.Parameters, true
		}
	}
	for name, executor := range doc.Executors {
		if position.InRange(executor.GetRange(), pos) {
			return fmt.Sprintf("the executor `%s`", name), executor.GetParameters(), true
		}
	}
	return "", nil, false
}

func describeParameter(parameter ast.Parameter, owner string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**%s** `%s`\n\n%s", parameter.GetName(), parameter.GetType(), owner)
	if enum, ok := parameter.(ast.EnumParameter); ok && len(enum.Enum) > 0 {
		fmt.Fprintf(&b, " One of `%s`.", strings.Join(enum.Enum, "`, `"))
	}
	if value, ok := defaultOf(parameter); ok {
		fmt.Fprintf(&b, " Default: `%s`.", value)
	} else if !parameter.IsOptional() {
		b.WriteString(" Required.")
	}
	if description := parameter.GetDescription(); description != "" {
		fmt.Fprintf(&b, "\n\n%s", description)
	}
	return b.String()
}
