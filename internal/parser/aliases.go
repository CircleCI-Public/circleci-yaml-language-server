package parser

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	ast2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
)

// parseAlias reads an entry of `commands`, `jobs` or `executors` whose value
// is a string, such as `build: orb/build`, as an alias. A value of any other
// type, such as a number or a mapping, is not one.
func (doc *YamlDocument) parseAlias(entryNode *sitter.Node) (ast2.Alias, bool) {
	keyNode, valueNode := doc.GetKeyValueNodes(entryNode)
	if keyNode == nil || valueNode == nil {
		return ast2.Alias{}, false
	}

	scalar := doc.scalarOf(valueNode, 0)
	switch scalar.Kind() {
	case "double_quote_scalar", "single_quote_scalar":
	case "plain_scalar":
		if child := GetFirstChild(scalar); child == nil || child.Kind() != "string_scalar" {
			return ast2.Alias{}, false
		}
	default:
		return ast2.Alias{}, false
	}

	return ast2.Alias{
		Name:        doc.getAttributeName(doc.GetNodeText(keyNode)),
		NameRange:   doc.NodeToRange(keyNode),
		Target:      doc.ScalarText(valueNode),
		TargetRange: doc.NodeToRange(valueNode),
		Range:       doc.NodeToRange(entryNode),
	}, true
}
