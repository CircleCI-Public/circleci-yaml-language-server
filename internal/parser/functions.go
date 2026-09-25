package parser

import (
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
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
