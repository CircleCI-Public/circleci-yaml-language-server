package parser

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
)

// parseFunctions reads the top-level `functions` block. Anything but a
// mapping is left alone, as the compiler leaves it: the key has long been
// free for other uses, such as holding YAML anchors.
func (doc *YamlDocument) parseFunctions(functionsNode *sitter.Node) {
	blockMappingNode := GetChildMapping(functionsNode)
	if blockMappingNode == nil {
		return
	}

	doc.iterateOnBlockMapping(blockMappingNode, func(child *sitter.Node) {
		keyNode, valueNode := doc.GetKeyValueNodes(child)
		if keyNode == nil {
			return
		}

		function := ast.Function{
			Alias:      doc.GetNodeText(keyNode),
			AliasRange: doc.NodeToRange(keyNode),
			Range:      doc.NodeToRange(child),
		}

		if valueNode != nil {
			function.ReferenceRange = doc.NodeToRange(valueNode)
			if scalar := GetFirstChild(valueNode); scalar != nil && isStringScalar(scalar.Kind()) {
				function.IsString = true
				function.Reference = doc.GetNodeText(valueNode)
			}
		}

		doc.Functions[function.Alias] = function
	})
}

func isStringScalar(kind string) bool {
	switch kind {
	case "plain_scalar", "double_quote_scalar", "single_quote_scalar":
		return true
	}
	return false
}

// FunctionForStep returns the declared function a step runs, and the command
// it names after a `/`, if any. A step is only a function step when its alias
// is declared, so that orbs and commands named the same way resolve as before.
func (doc *YamlDocument) FunctionForStep(stepName string) (ast.Function, string, bool) {
	if function, ok := doc.Functions[stepName]; ok {
		return function, "", true
	}

	alias, command, found := strings.Cut(stepName, "/")
	if !found {
		return ast.Function{}, "", false
	}

	function, ok := doc.Functions[alias]
	return function, command, ok
}

// LookUpFunction finds a declaration in the functions catalog. The function
// is nil when none is published under the declaration's path, and the
// descriptor is nil when that function has no such version. An error means
// the catalog couldn't be read, as on a host that doesn't serve it, and says
// nothing about the declaration.
func (doc *YamlDocument) LookUpFunction(function ast.Function, c *cache.Cache) (*circleci.FunctionPackage, *circleci.FunctionDescriptor, error) {
	path, version, found := strings.Cut(function.Reference, "@")
	if !function.IsString || !found {
		return nil, nil, fmt.Errorf("function %s has no version", function.Alias)
	}

	client := doc.Context.V3Client()
	published, err := c.Functions.Function(client, path)
	if err != nil || published == nil {
		return nil, nil, err
	}

	for _, candidate := range published.Versions {
		if candidate.Version != version {
			continue
		}

		descriptor, err := c.Functions.Descriptor(client, candidate)
		return published, descriptor, err
	}

	return published, nil, nil
}
