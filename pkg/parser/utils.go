package parser

import (
	"fmt"
	"strconv"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	ymlgrammar "github.com/smacker/go-tree-sitter/yaml"
	"go.lsp.dev/protocol"
)

func GetRootNode(content []byte) *sitter.Node {
	parser := sitter.NewParser()
	parser.SetLanguage(ymlgrammar.GetLanguage())

	tree := parser.Parse(nil, content)

	return tree.RootNode()
}

func GetChildOfType(node *sitter.Node, typeName string) *sitter.Node {
	if node == nil {
		return nil
	}
	for i := 0; uint32(i) < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Type() == typeName {
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
		if node.Child(0).Type() == "comment" || node.Child(0).Type() == "anchor" {
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
	if documentNode.Type() != "document" {
		return nil
	}
	blockNode := GetChildOfType(documentNode, "block_node")
	if blockNode == nil {
		return nil
	}

	return GetChildOfType(blockNode, "block_mapping")
}

func (doc *YamlDocument) GetNodeText(node *sitter.Node) string {
	res := doc.GetRawNodeText(node)

	if strings.HasPrefix(res, "\"") && strings.HasSuffix(res, "\"") {
		res = strings.Trim(res, "\"")
	} else if strings.HasPrefix(res, "'") && strings.HasSuffix(res, "'") {
		res = strings.Trim(res, "'")
	}

	res = strings.TrimPrefix(res, "|\n")
	res = strings.TrimPrefix(res, ">-\n")
	res = strings.TrimPrefix(res, ">\n")
	res = strings.TrimSpace(res)

	return res
}

func (doc *YamlDocument) GetRawNodeText(node *sitter.Node) string {
	if node == nil {
		return ""
	}
	res := string(doc.Content[node.StartByte():node.EndByte()])
	return res
}

type TextAndRange struct {
	Text  string
	Range protocol.Range
}

func (doc *YamlDocument) getNodeTextArray(valueNode *sitter.Node) []string {
	res := doc.getNodeTextArrayWithRange(valueNode)
	texts := make([]string, len(res))
	for i, textAndRange := range res {
		texts[i] = textAndRange.Text
	}
	return texts
}

func (doc *YamlDocument) getNodeTextArrayWithRange(valueNode *sitter.Node) []TextAndRange {
	// valueNode is block_node which has a block_sequence child
	blockSequenceNode := GetChildOfType(valueNode, "block_sequence")
	texts := make([]TextAndRange, 0)

	if blockSequenceNode == nil {
		blockSequenceNode = GetChildSequence(valueNode)
		if blockSequenceNode == nil {
			return texts
		}
	}

	iterateOnBlockSequence(blockSequenceNode, func(child *sitter.Node) {
		getText := func(node *sitter.Node) TextAndRange {
			if alias := GetChildOfType(node, "alias"); alias != nil {
				anchor, ok := doc.YamlAnchors[strings.TrimLeft(doc.GetNodeText(alias), "*")]
				if !ok {
					return TextAndRange{Text: "wtf"}
				}
				anchorValueNode := GetFirstChild(anchor.ValueNode)
				text := doc.GetNodeText(anchorValueNode)
				return TextAndRange{Text: text, Range: NodeToRange(anchorValueNode)}
			} else {

				return TextAndRange{Text: doc.GetNodeText(node), Range: NodeToRange(node)}
			}
		}

		// If blockSequence is a flow_sequence, then the child is
		// directly a flow_node
		if child.Type() == "flow_node" {
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
		if child.Type() == "block_mapping_pair" || child.Type() == "flow_pair" {
			keyNode, valueNode := doc.GetKeyValueNodes(child)

			if keyNode != nil && valueNode != nil {
				dictionary[doc.GetNodeText(keyNode)] = doc.GetNodeText(valueNode)
			}
		}
	})

	return dictionary
}

func (doc *YamlDocument) parseDescription(descriptionNode *sitter.Node) string {
	return doc.GetNodeText(descriptionNode)
}

func (doc *YamlDocument) GetKeyValueNodes(node *sitter.Node) (keyNode *sitter.Node, valueNode *sitter.Node) {
	if node != nil && (node.Type() == "block_mapping_pair" || node.Type() == "flow_pair") {
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
	if blockMappingNode == nil || (blockMappingNode.Type() != "block_mapping" && blockMappingNode.Type() != "flow_mapping") {
		return
	}

	// Save keys that are mapped on, to support merge keys (<<: *someAlias)
	// For common keys between a map using a merge key & the mapped alias
	// the merged value does NOT override the parent block-mapping defined key
	// For this reason, it is important to know which keys have already been evaluated or not
	mappedKeys := map[string]bool{}
	mergeKeys := map[string]bool{}

	for i := 0; uint32(i) < blockMappingNode.ChildCount(); i++ {
		child := blockMappingNode.Child(i)

		if child.Type() == "comment" {
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

		anchorValue := GetFirstChild(anchor.ValueNode)

		// Recursively call iterateOnBlockMapping to handle merged block that contain merged blocks themselves
		doc.iterateOnBlockMapping(
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
		)
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
	if child.Type() == "alias" {
		txt := doc.GetNodeText(child)

		return []string{txt[1:]}
	}

	// List of aliases; return all of em dude
	// example: <<: [*alias1, *alias2, ..., *aliasN]
	if child.Type() == "flow_sequence" {
		names := []string{}

		for i := 0; uint32(i) < child.ChildCount(); i++ {
			names = append(names, extractMergeAnchorNames(child.Child(i), doc)...)
		}

		return names
	}

	return []string{}
}

func iterateOnBlockSequence(blockSequenceNode *sitter.Node, fn func(child *sitter.Node)) {
	if blockSequenceNode == nil ||
		(blockSequenceNode.Type() != "block_sequence" && blockSequenceNode.Type() != "flow_sequence") {
		return
	}
	for i := 0; uint32(i) < blockSequenceNode.ChildCount(); i++ {
		child := blockSequenceNode.Child(i)

		if child.Type() == "comment" {
			continue
		}

		fn(child)
	}
}

func ExecQuery(node *sitter.Node, query string, fn func(match *sitter.QueryMatch)) error {
	pattern := []byte(query)
	queryTreeSitter, err := sitter.NewQuery(pattern, ymlgrammar.GetLanguage())
	if err != nil {
		return err
	}

	cursor := sitter.NewQueryCursor()
	cursor.Exec(queryTreeSitter, node)
	anchorMatches, found := cursor.NextMatch()
	for found {
		fn(anchorMatches)
		anchorMatches, found = cursor.NextMatch()
	}

	return nil
}

func FindDeepestNode(rootNode *sitter.Node, content []byte, toFind []string) (*sitter.Node, error) {
	if len(toFind) == 0 {
		return rootNode, nil
	}

	iterator := sitter.NewIterator(rootNode, sitter.DFSMode)
	node, err := iterator.Next()

	for err == nil {
		if intValue, err := strconv.Atoi(toFind[0]); err == nil {
			if node.Type() == "block_sequence" {
				if node.ChildCount() < uint32(intValue+1) {
					return nil, fmt.Errorf("index out of range: trying to access %d in array of size %d", intValue, node.ChildCount())
				}

				childNode := node.Child((intValue))
				return FindDeepestNode(childNode, content, toFind[1:])
			}
		}
		if node.Type() == "block_mapping_pair" {
			if key := node.ChildByFieldName("key"); string(content[key.StartByte():key.EndByte()]) == toFind[0] {
				return FindDeepestNode(node, content, toFind[1:])
			}
		}
		node, err = iterator.Next()
	}

	return node, fmt.Errorf("not found")
}

func NodeToRange(node *sitter.Node) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{Line: node.StartPoint().Row, Character: node.StartPoint().Column},
		End:   protocol.Position{Line: node.EndPoint().Row, Character: node.EndPoint().Column},
	}
}
