package parser

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
)

func (doc *YamlDocument) parseEnvs(node *sitter.Node) ast.Environment {
	if node == nil || node.Kind() != "block_node" {
		return ast.Environment{}
	}

	blockMapping := node.Child(0)

	if blockMapping == nil || blockMapping.Kind() != "block_mapping" {
		return ast.Environment{}
	}

	return ast.Environment{
		Range:     doc.NodeToRange(node),
		Variables: doc.parseEnvVariables(blockMapping),
	}
}

func (doc *YamlDocument) parseEnvVariables(blockMapping *sitter.Node) []ast.EnvironmentVariable {
	variables := []ast.EnvironmentVariable{}

	doc.iterateOnBlockMapping(
		blockMapping,
		func(child *sitter.Node) {
			keyNode, _ := doc.GetKeyValueNodes(child)
			if keyNode == nil {
				return
			}

			variables = append(variables, ast.EnvironmentVariable{
				Name:      doc.GetNodeText(keyNode),
				Range:     doc.NodeToRange(child),
				NameRange: doc.NodeToRange(keyNode),
			})
		},
	)

	return variables
}
