package hover

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/stepoutputs"
)

// StepOutput is the hover for a reference to a function step's output, such
// as `${{ steps.build.outputs.digest }}`: the type and description the
// function declares for the output, and whether it is secret.
func StepOutput(ctx context.Context, doc yamlparser.YamlDocument, c *cache.Cache, pos protocol.Position) (string, bool) {
	lineStart := position.ToIndex(protocol.Position{Line: pos.Line}, doc.Content)
	lineEnd := bytes.IndexByte(doc.Content[lineStart:], '\n')
	if lineEnd == -1 {
		lineEnd = len(doc.Content) - lineStart
	}
	line := doc.Content[lineStart : lineStart+lineEnd]
	column := int(pos.Character)

	for _, match := range stepoutputs.Pattern.FindAllSubmatchIndex(line, -1) {
		if column < match[0] || column > match[1] {
			continue
		}
		id, name := string(line[match[2]:match[3]]), string(line[match[4]:match[5]])

		steps, ok := doc.OutputStepsAt(pos)
		if !ok {
			return "", false
		}
		i := slices.IndexFunc(steps, func(step yamlparser.OutputStep) bool { return step.ID == id })
		if i == -1 {
			return "", false
		}
		_, descriptor, err := doc.LookUpFunction(ctx, steps[i].Function, c)
		if err != nil || descriptor == nil {
			return "", false
		}
		outputs := descriptor.DeclaredOutputs()
		j := slices.IndexFunc(outputs, func(output circleci.FunctionOutput) bool { return output.Name == name })
		if j == -1 {
			return "", false
		}
		return describeOutput(steps[i].Step.Name, outputs[j]), true
	}
	return "", false
}

func describeOutput(step string, output circleci.FunctionOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**%s** output of **%s**\n\n(%s", output.Name, step, output.Type)
	if output.Secret {
		b.WriteString(", secret")
	}
	b.WriteString(")")
	if output.Description != "" {
		fmt.Fprintf(&b, ": %s", output.Description)
	}
	return b.String()
}
