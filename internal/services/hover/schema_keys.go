package hover

import (
	"strconv"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

// SchemaKey is the hover for a key: what the schema says of it, such as
// resource_class's description. A value gets none.
func SchemaKey(doc yamlparser.YamlDocument, _ *cache.Cache, pos protocol.Position) (string, bool) {
	path, ok := keyPathAt(doc, pos)
	if !ok {
		return "", false
	}
	text := yamlparser.SchemaKeyDescription(path)
	return text, text != ""
}

// keyPathAt is the path to the key the cursor is on, such as
// jobs.build.steps.0.run, with a list item's index in place of the item.
func keyPathAt(doc yamlparser.YamlDocument, pos protocol.Position) ([]string, bool) {
	if doc.RootNode == nil {
		return nil, false
	}
	offset := uint(position.ToIndex(pos, doc.Content))
	node := doc.RootNode.NamedDescendantForByteRange(offset, offset)

	pair := enclosingPair(node)
	if pair == nil {
		return nil, false
	}
	key := pair.ChildByFieldName("key")
	if key == nil || offset < key.StartByte() || offset > key.EndByte() {
		return nil, false
	}

	var path []string
	for n := pair; n != nil; n = n.Parent() {
		switch n.Kind() {
		case "block_mapping_pair", "flow_pair":
			k := n.ChildByFieldName("key")
			if k == nil {
				return nil, false
			}
			path = append(path, keyName(doc.GetNodeText(k)))
		case "block_sequence_item", "flow_node":
			if index, ok := itemIndex(n); ok {
				path = append(path, strconv.Itoa(index))
			}
		}
	}

	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path, true
}

// enclosingPair is the closest mapping pair holding node, or nil.
func enclosingPair(node *sitter.Node) *sitter.Node {
	for n := node; n != nil; n = n.Parent() {
		if kind := n.Kind(); kind == "block_mapping_pair" || kind == "flow_pair" {
			return n
		}
	}
	return nil
}

// itemIndex is the position of node in the list it is an item of: a block
// sequence's item, or a flow node directly inside a flow sequence.
func itemIndex(node *sitter.Node) (int, bool) {
	parent := node.Parent()
	if parent == nil {
		return 0, false
	}
	switch {
	case node.Kind() == "block_sequence_item" && parent.Kind() == "block_sequence":
	case node.Kind() == "flow_node" && parent.Kind() == "flow_sequence":
	default:
		return 0, false
	}

	index := 0
	for i := uint(0); i < parent.NamedChildCount(); i++ {
		sibling := parent.NamedChild(i)
		if sibling.Id() == node.Id() {
			return index, true
		}
		if sibling.Kind() == node.Kind() {
			index++
		}
	}
	return 0, false
}

func keyName(text string) string {
	if len(text) >= 2 && (text[0] == '"' || text[0] == '\'') && text[len(text)-1] == text[0] {
		return text[1 : len(text)-1]
	}
	return strings.TrimSpace(text)
}
