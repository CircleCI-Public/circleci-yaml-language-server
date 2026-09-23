package parser

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func (doc *YamlDocument) parseEnvs(node *sitter.Node) ast.Environment {
	if node == nil || node.Kind() != "block_node" {
		return ast.Environment{}
	}

	blockMapping := node.Child(0)

	if blockMapping == nil || blockMapping.Kind() != "block_mapping" {
		return ast.Environment{}
	}

	keys := []string{}

	doc.iterateOnBlockMapping(
		blockMapping,
		func(child *sitter.Node) {
			keyNode, _ := doc.GetKeyValueNodes(child)

			keys = append(keys, doc.GetNodeText(keyNode))
		},
	)

	return ast.Environment{
		Range: doc.NodeToRange(node),
		Keys:  keys,
	}
}
