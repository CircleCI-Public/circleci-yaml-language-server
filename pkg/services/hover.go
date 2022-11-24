package languageservice

import (
	"fmt"

	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	utils "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

func Hover(params protocol.HoverParams, cache *utils.Cache) (protocol.Hover, error) {
	doc, err := yamlparser.GetParsedYAMLWithCache(params.TextDocument.URI, cache)
	if err != nil {
		return protocol.Hover{}, nil
	}

	if utils.PosInRange(doc.VersionRange, params.Position) && doc.Version < 2.1 {
		return protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.PlainText,
				Value: "Circle CI Config Helper is not available for this version. (Supported: 2.1)",
			},
		}, nil
	}

	return protocol.Hover{}, fmt.Errorf("No hover")
}

func GetPathFromVisitedNodes(visitedNodes []*sitter.Node, doc yamlparser.YamlDocument) []string {
	var path []string
	if len(visitedNodes) == 0 {
		return path
	}

	for _, node := range visitedNodes[1:] {
		switch node.Type() {
		case "block_mapping_pair":
			key := node.ChildByFieldName("key")
			name := string(doc.Content[key.StartByte():key.EndByte()])
			path = append(path, name)
		case "block_sequence_item":
			flow_node := yamlparser.GetChildOfType(node, "flow_node")
			if flow_node != nil {
				name := string(doc.Content[flow_node.StartByte():flow_node.EndByte()])
				path = append(path, name)
			}
		}
	}

	return path
}
