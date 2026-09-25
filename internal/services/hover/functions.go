package hover

import (
	"fmt"
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
func FunctionStep(doc yamlparser.YamlDocument, c *cache.Cache, pos protocol.Position) (string, bool) {
	step, ok := namedStepAt(pos, doc.Jobs, doc.Commands)
	if !ok {
		return "", false
	}
	function, command, ok := doc.FunctionForStep(step.Name)
	if !ok {
		return "", false
	}

	_, descriptor, err := doc.LookUpFunction(function, c)
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
// description and flags of the version it declares.
func FunctionDeclaration(doc yamlparser.YamlDocument, c *cache.Cache, pos protocol.Position) (string, bool) {
	function, ok := declarationAt(doc, pos)
	if !ok {
		return "", false
	}

	_, descriptor, err := doc.LookUpFunction(function, c)
	if err != nil || descriptor == nil {
		return "", false
	}
	return describeFunction(function.Alias, "function", descriptor.Description, descriptor.Flags), true
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
