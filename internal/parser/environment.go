package parser

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/yamltree"
	sitter "github.com/odvcencio/gotreesitter"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
)

func (doc *YamlDocument) parseEnvs(node *sitter.Node) ast.Environment {
	if node == nil || node.Type(yamltree.Language) != "block_node" {
		return ast.Environment{}
	}

	blockMapping := node.Child(0)

	if blockMapping == nil || blockMapping.Type(yamltree.Language) != "block_mapping" {
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
