package languageservice

import (
	"fmt"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/services/hover"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

func Hover(params protocol.HoverParams, cache *cache.Cache, context *session.Settings) (protocol.Hover, error) {
	doc, err := parser.ParseFromUriWithCache(params.TextDocument.URI, cache, context)
	if err != nil {
		return protocol.Hover{}, nil
	}
	defer doc.Close()

	if position.InRange(doc.VersionRange, params.Position) && doc.Version < 2.1 {
		return protocol.Hover{
			Contents: &protocol.MarkupContent{
				Kind:  protocol.MarkupKindPlainText,
				Value: "Circle CI Config Helper is not available for this version. (Supported: 2.1)",
			},
		}, nil
	}

	if text, ok := hover.Step(doc, cache, params.Position); ok {
		return protocol.Hover{
			Contents: &protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: text},
		}, nil
	}

	return protocol.Hover{}, fmt.Errorf("no hover")
}

func GetPathFromVisitedNodes(visitedNodes []*sitter.Node, doc parser.YamlDocument) []string {
	var path []string
	if len(visitedNodes) == 0 {
		return path
	}

	for _, node := range visitedNodes[1:] {
		switch node.Kind() {
		case "block_mapping_pair":
			// A pair whose key is yet to be typed, as `: value` is, has no
			// name to go in the path.
			key := node.ChildByFieldName("key")
			if key == nil {
				continue
			}
			name := string(doc.Content[key.StartByte():key.EndByte()])
			path = append(path, name)
		case "block_sequence_item":
			flow_node := parser.GetChildOfType(node, "flow_node")
			if flow_node != nil {
				name := string(doc.Content[flow_node.StartByte():flow_node.EndByte()])
				path = append(path, name)
			}
		}
	}

	return path
}
