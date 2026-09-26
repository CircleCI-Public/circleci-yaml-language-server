package validate

import (
	"fmt"
	"regexp"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/diagnostic"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/paramref"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/yamltree"
)

// ValidateRegexes reports the regular expressions the compiler can't compile:
// a logic statement's `matches` pattern, and a branch or tag filter written
// as `/.../`. Like the compiler, it doesn't accept lookarounds or
// backreferences.
func (val Validate) ValidateRegexes() {
	for node := range yamltree.Walk(val.Doc.RootNode) {
		if !isPair(node) || val.Doc.IsUnderUnreadTopLevelKey(node) {
			continue
		}

		switch val.pairKey(node) {
		case "pattern":
			if val.pairKey(enclosingPair(node)) == "matches" {
				for _, value := range val.pairValues(node) {
					val.checkRegex(value, val.Doc.ScalarText(value))
				}
			}
		case "only", "ignore":
			if filter := val.pairKey(enclosingPair(node)); filter == "branches" || filter == "tags" {
				for _, value := range val.pairValues(node) {
					text := val.Doc.ScalarText(value)
					if len(text) > 1 && strings.HasPrefix(text, "/") && strings.HasSuffix(text, "/") {
						val.checkRegex(value, text[1:len(text)-1])
					}
				}
			}
		}
	}
}

func (val Validate) checkRegex(node *sitter.Node, pattern string) {
	if paramref.ContainsReference(pattern) {
		return
	}
	if _, err := regexp.Compile(pattern); err != nil {
		val.addDiagnostic(diagnostic.ErrorFromNode(node, fmt.Sprintf("Invalid regular expression: %s", pattern)))
	}
}

func isPair(node *sitter.Node) bool {
	return node != nil && (node.Kind() == "block_mapping_pair" || node.Kind() == "flow_pair")
}

// enclosingPair returns the pair whose value holds pair.
func enclosingPair(pair *sitter.Node) *sitter.Node {
	for node := pair.Parent(); node != nil; node = node.Parent() {
		if isPair(node) {
			return node
		}
	}
	return nil
}

func (val Validate) pairKey(pair *sitter.Node) string {
	if pair == nil {
		return ""
	}
	key, _ := val.Doc.GetKeyValueNodes(pair)
	return val.Doc.GetNodeText(key)
}

// pairValues returns the value of pair, or each of its items when it's a
// sequence.
func (val Validate) pairValues(pair *sitter.Node) []*sitter.Node {
	_, value := val.Doc.GetKeyValueNodes(pair)
	if value == nil {
		return nil
	}
	sequence := parser.GetChildSequence(value)
	if sequence == nil {
		return []*sitter.Node{value}
	}

	values := []*sitter.Node{}
	for i := uint(0); i < sequence.NamedChildCount(); i++ {
		item := sequence.NamedChild(i)
		if item.Kind() == "block_sequence_item" {
			item = item.NamedChild(0)
		}
		if item != nil && item.Kind() != "comment" {
			values = append(values, item)
		}
	}
	return values
}
