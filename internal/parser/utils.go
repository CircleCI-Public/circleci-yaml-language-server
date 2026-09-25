package parser

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"
	"gopkg.in/yaml.v3"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/yamltree"
)

func GetChildOfType(node *sitter.Node, typeName string) *sitter.Node {
	if node == nil {
		return nil
	}
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == typeName {
			return child
		}
	}
	return nil
}

func GetFirstChild(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	if node.ChildCount() > 0 {
		if node.Child(0).Kind() == "comment" || node.Child(0).Kind() == "anchor" {
			return node.Child(1)
		}
		return node.Child(0)
	}
	return nil
}

func GetChildMapping(node *sitter.Node) *sitter.Node {
	blockMappingNode := GetChildOfType(node, "block_mapping")

	if blockMappingNode != nil {
		return blockMappingNode
	}

	return GetChildOfType(node, "flow_mapping")
}

func GetChildSequence(node *sitter.Node) *sitter.Node {
	blockMappingNode := GetChildOfType(node, "block_sequence")

	if blockMappingNode != nil {
		return blockMappingNode
	}

	return GetChildOfType(node, "flow_sequence")
}

func GetBlockMappingNode(streamNode *sitter.Node) *sitter.Node {
	documentNode := GetChildOfType(streamNode, "document")
	if documentNode != nil && documentNode.Kind() != "document" {
		return nil
	}
	blockNode := GetChildOfType(documentNode, "block_node")
	if blockNode == nil {
		return nil
	}

	return GetChildOfType(blockNode, "block_mapping")
}

func (doc *YamlDocument) GetNodeTextWithRange(node *sitter.Node) ast.TextAndRange {
	if node == nil {
		return ast.TextAndRange{Text: "", Range: protocol.Range{}}
	}

	res := doc.GetRawNodeText(doc.scalarOf(node, 0))

	if strings.HasPrefix(res, "\"") && strings.HasSuffix(res, "\"") {
		res = strings.Trim(res, "\"")
	} else if strings.HasPrefix(res, "'") && strings.HasSuffix(res, "'") {
		res = strings.Trim(res, "'")
	}

	res = strings.TrimPrefix(res, "|\n")
	res = strings.TrimPrefix(res, ">-\n")
	res = strings.TrimPrefix(res, ">\n")
	res = strings.TrimSpace(res)

	return ast.TextAndRange{
		Text:  res,
		Range: doc.NodeToRange(node),
	}
}

var scalarKinds = []string{"plain_scalar", "double_quote_scalar", "single_quote_scalar", "block_scalar"}

// scalarOf returns the scalar a value node holds, without the anchor or tag
// written before it, and the anchored scalar an alias names. Any other node is
// returned as it is.
func (doc *YamlDocument) scalarOf(node *sitter.Node, depth int) *sitter.Node {
	if node.Kind() != "flow_node" && node.Kind() != "block_node" {
		return node
	}

	var scalar, alias *sitter.Node
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		switch {
		case child.Kind() == "alias":
			alias = child
		case slices.Contains(scalarKinds, child.Kind()):
			scalar = child
		case child.Kind() != "anchor" && child.Kind() != "tag" && child.Kind() != "comment":
			return node
		}
	}

	if scalar != nil {
		return scalar
	}
	if alias != nil && depth < 10 {
		anchor, ok := doc.YamlAnchors[strings.TrimPrefix(doc.GetRawNodeText(alias), "*")]
		if ok && anchor.ValueNode != nil {
			if resolved := doc.scalarOf(anchor.ValueNode, depth+1); slices.Contains(scalarKinds, resolved.Kind()) {
				return resolved
			}
		}
	}
	return node
}

func (doc *YamlDocument) GetNodeText(node *sitter.Node) string {
	return doc.GetNodeTextWithRange(node).Text
}

func (doc *YamlDocument) GetRawNodeText(node *sitter.Node) string {
	if node == nil {
		return ""
	}
	res := string(doc.Content[node.StartByte():node.EndByte()])
	return res
}

func (doc *YamlDocument) getNodeTextArray(valueNode *sitter.Node) []string {
	res := doc.getNodeTextArrayWithRange(valueNode)
	texts := make([]string, len(res))
	for i, textAndRange := range res {
		texts[i] = textAndRange.Text
	}
	return texts
}

func (doc *YamlDocument) getNodeTextArrayWithRange(valueNode *sitter.Node) []ast.TextAndRange {
	// valueNode is block_node which has a block_sequence child
	blockSequenceNode := GetChildSequence(valueNode)
	texts := make([]ast.TextAndRange, 0)

	if blockSequenceNode == nil {
		return texts
	}

	iterateOnBlockSequence(blockSequenceNode, func(child *sitter.Node) {
		getText := func(node *sitter.Node) ast.TextAndRange {
			if alias := GetChildOfType(node, "alias"); alias != nil {
				anchor, ok := doc.YamlAnchors[strings.TrimLeft(doc.GetNodeText(alias), "*")]
				if !ok {
					return ast.TextAndRange{Text: ""}
				}
				anchorValueNode := GetFirstChild(anchor.ValueNode)
				text := doc.GetNodeText(anchorValueNode)
				return ast.TextAndRange{Text: text, Range: doc.NodeToRange(anchorValueNode)}
			} else {
				return ast.TextAndRange{Text: doc.GetNodeText(node), Range: doc.NodeToRange(node)}
			}
		}

		// If blockSequence is a flow_sequence, then the child is
		// directly a flow_node
		if child.Kind() == "flow_node" {
			texts = append(texts, getText(child))
		} else {
			// But if the blockSequence is a block_sequence, then the child is
			// a block_sequence_item
			element := GetChildOfType(child, "flow_node")
			hyphenNode := child.Child(0)
			if element != nil {
				texts = append(texts, getText(element))
			} else if hyphenNode != nil {
				texts = append(texts, getText(hyphenNode.NextSibling()))
			}
		}
	})

	return texts
}

func (doc *YamlDocument) getNodeTextArrayOrText(valueNode *sitter.Node) []string {
	textArray := doc.getNodeTextArray(valueNode)
	if len(textArray) == 0 {
		return []string{doc.GetNodeText(valueNode)}
	}
	return textArray
}

func (doc *YamlDocument) parseDictionary(valueNode *sitter.Node) map[string]string {
	dictionary := make(map[string]string)

	doc.iterateOnBlockMapping(valueNode, func(child *sitter.Node) {
		if child.Kind() == "block_mapping_pair" || child.Kind() == "flow_pair" {
			keyNode, valueNode := doc.GetKeyValueNodes(child)

			if keyNode != nil && valueNode != nil {
				dictionary[doc.GetNodeText(keyNode)] = doc.GetNodeText(valueNode)
			}
		}
	})

	return dictionary
}

// parseDescription is a description's text, as YAML reads it: without the
// quotes of a quoted string, and with a block scalar's indentation and folding
// applied. When YAML can't read it as text, it's the text GetNodeText gives.
func (doc *YamlDocument) parseDescription(descriptionNode *sitter.Node) string {
	if descriptionNode == nil {
		return ""
	}

	var text string
	if err := yaml.Unmarshal([]byte(doc.GetRawNodeText(doc.scalarOf(descriptionNode, 0))), &text); err != nil {
		return doc.GetNodeText(descriptionNode)
	}
	return strings.TrimRight(text, "\n")
}

func (doc *YamlDocument) GetKeyValueNodes(node *sitter.Node) (keyNode *sitter.Node, valueNode *sitter.Node) {
	if node != nil && (node.Kind() == "block_mapping_pair" || node.Kind() == "flow_pair") {
		keyNode = node.ChildByFieldName("key")
		valueNode = node.ChildByFieldName("value")

		aliasNode := GetChildOfType(valueNode, "alias")
		if aliasNode != nil {
			aliasNode = GetChildOfType(aliasNode, "alias_name")
			valueName := doc.GetNodeText(aliasNode)
			anchor, ok := doc.YamlAnchors[valueName]
			if ok {
				valueNode = anchor.ValueNode
			}
		}
	}
	return
}

func (doc *YamlDocument) iterateOnBlockMapping(blockMappingNode *sitter.Node, fn func(child *sitter.Node)) {
	doc.iterateOnMergedBlockMapping(blockMappingNode, fn, map[string]bool{})
}

// iterateOnMergedBlockMapping is iterateOnBlockMapping for a mapping reached
// through the merge keys of the anchors in merging. An anchor already among
// them is not merged again: an anchor that merges itself, directly or through
// others, would otherwise be merged forever.
func (doc *YamlDocument) iterateOnMergedBlockMapping(blockMappingNode *sitter.Node, fn func(child *sitter.Node), merging map[string]bool) {
	if blockMappingNode == nil || (blockMappingNode.Kind() != "block_mapping" && blockMappingNode.Kind() != "flow_mapping") {
		return
	}

	// Save keys that are mapped on, to support merge keys (<<: *someAlias)
	// For common keys between a map using a merge key & the mapped alias
	// the merged value does NOT override the parent block-mapping defined key
	// For this reason, it is important to know which keys have already been evaluated or not
	mappedKeys := map[string]bool{}
	mergeKeys := map[string]bool{}

	for i := uint(0); i < blockMappingNode.ChildCount(); i++ {
		child := blockMappingNode.Child(i)

		if child.Kind() == "comment" {
			continue
		}

		keyNode := child.ChildByFieldName("key")
		keyText := doc.GetNodeText(keyNode)

		// When a merge key is encountered, skip it.
		// it will be handled after other keys, to avoid potential falsy override
		if keyText == "<<" {
			valueNode := child.ChildByFieldName("value")
			anchorsToMerge := extractMergeAnchorNames(valueNode, doc)

			for _, anchorName := range anchorsToMerge {
				mergeKeys[anchorName] = true
			}

			continue
		}

		fn(child)
		mappedKeys[keyText] = true
	}

	for anchorName := range mergeKeys {
		anchor, ok := doc.YamlAnchors[anchorName]

		if !ok || anchor.ValueNode == nil {
			return
		}

		if merging[anchorName] {
			continue
		}

		anchorValue := GetFirstChild(anchor.ValueNode)

		// Recursively call iterateOnBlockMapping to handle merged block that contain merged blocks themselves
		merging[anchorName] = true
		doc.iterateOnMergedBlockMapping(
			anchorValue,
			func(child *sitter.Node) {
				keyNode, _ := doc.GetKeyValueNodes(child)
				keyName := doc.GetNodeText(keyNode)

				// On each key defined by the to-merge block;
				// check if the key has already been evaluated by the
				// parent definition or not
				if mappedKeys[keyName] {
					return
				}

				// If it hasn't, call the original callback and store the key too
				fn(child)
				mappedKeys[keyName] = true
			},
			merging,
		)
		delete(merging, anchorName)
	}

}

func extractMergeAnchorNames(node *sitter.Node, doc *YamlDocument) []string {
	if node == nil {
		return nil
	}

	child := node.Child(0)

	if child == nil {
		return nil
	}

	// One alias; just return the alias name
	// example: <<: *myAlias
	if child.Kind() == "alias" {
		txt := doc.GetNodeText(child)

		return []string{txt[1:]}
	}

	// List of aliases; return all of em dude
	// example: <<: [*alias1, *alias2, ..., *aliasN]
	if child.Kind() == "flow_sequence" {
		names := []string{}

		for i := uint(0); i < child.ChildCount(); i++ {
			names = append(names, extractMergeAnchorNames(child.Child(i), doc)...)
		}

		return names
	}

	return []string{}
}

func iterateOnBlockSequence(blockSequenceNode *sitter.Node, fn func(child *sitter.Node)) {
	if blockSequenceNode == nil ||
		(blockSequenceNode.Kind() != "block_sequence" && blockSequenceNode.Kind() != "flow_sequence") {
		return
	}
	for i := uint(0); i < blockSequenceNode.ChildCount(); i++ {
		child := blockSequenceNode.Child(i)

		if child.Kind() == "comment" {
			continue
		}

		fn(child)
	}
}

// FindDeepestNode returns the node at the path toFind, of mapping keys and
// sequence indexes, below rootNode.
func FindDeepestNode(rootNode *sitter.Node, content []byte, toFind []string) (*sitter.Node, error) {
	if len(toFind) == 0 {
		return rootNode, nil
	}

	if child, err := findChildNode(rootNode, content, toFind[0]); err != nil {
		return nil, err
	} else if child != nil {
		return FindDeepestNode(child, content, toFind[1:])
	}

	// A value merged in with `<<: *anchor` isn't below the key it's merged
	// into, so fall back to the first match anywhere below.
	for node := range yamltree.Walk(rootNode) {
		if intValue, err := strconv.Atoi(toFind[0]); err == nil && intValue >= 0 {
			if node.Kind() == "block_sequence" {
				if node.ChildCount() < uint(intValue+1) {
					return nil, fmt.Errorf("index out of range: trying to access %d in array of size %d", intValue, node.ChildCount())
				}

				childNode := node.Child(uint(intValue))
				return FindDeepestNode(childNode, content, toFind[1:])
			}
		}
		if node.Kind() == "block_mapping_pair" {
			if key := node.ChildByFieldName("key"); string(content[key.StartByte():key.EndByte()]) == toFind[0] {
				return FindDeepestNode(node, content, toFind[1:])
			}
		}
	}

	return nil, fmt.Errorf("not found")
}

// findChildNode returns the pair for key, or the item at index key, of the
// mapping or sequence that node holds. It doesn't look inside the other
// pairs and items, whose own keys would otherwise be matched first when
// they come earlier in the file, as a workflow's `jobs:` does before the
// top-level one.
func findChildNode(node *sitter.Node, content []byte, key string) (*sitter.Node, error) {
	index, err := strconv.Atoi(key)
	isIndex := err == nil && index >= 0

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		switch child.Kind() {
		case "block_mapping_pair":
			if keyNode := child.ChildByFieldName("key"); !isIndex && keyNode != nil &&
				string(content[keyNode.StartByte():keyNode.EndByte()]) == key {
				return child, nil
			}
			continue
		case "block_sequence":
			if !isIndex {
				continue
			}
			if child.ChildCount() < uint(index+1) {
				return nil, fmt.Errorf("index out of range: trying to access %d in array of size %d", index, child.ChildCount())
			}
			return child.Child(uint(index)), nil
		}

		if found, err := findChildNode(child, content, key); found != nil || err != nil {
			return found, err
		}
	}

	return nil, nil
}

func (doc *YamlDocument) NodeToRange(node *sitter.Node) protocol.Range {
	if node == nil {
		return protocol.Range{}
	}
	return position.AddOffsetToRange(protocol.Range{
		Start: protocol.Position{
			Line:      position.Start(node).Line,
			Character: position.Start(node).Character,
		},
		End: protocol.Position{
			Line:      position.End(node).Line,
			Character: position.End(node).Character,
		},
	}, doc.Offset)
}

// unquotedRange is the range of a single-line scalar's text, inside its quotes
// if it has them: the range GetNodeText's result occupies.
func (doc *YamlDocument) unquotedRange(node *sitter.Node) protocol.Range {
	rng := doc.NodeToRange(node)
	raw := doc.GetRawNodeText(node)

	quoted := len(raw) >= 2 && (raw[0] == '"' || raw[0] == '\'') && raw[len(raw)-1] == raw[0]
	if quoted && rng.Start.Line == rng.End.Line {
		rng.Start.Character++
		rng.End.Character--
	}

	return rng
}
