package position

import (
	"fmt"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/yamltree"
	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"
)

func NodeAt(rootNode *sitter.Node, pos protocol.Position) (*sitter.Node, []*sitter.Node, error) {
	listOfCandidates := make([]*sitter.Node, 0)

	for node := range yamltree.Walk(rootNode) {
		if InRange(protocol.Range{Start: Start(node), End: End(node)}, pos) {
			listOfCandidates = append(listOfCandidates, node)
		}
	}

	if len(listOfCandidates) == 0 {
		return nil, listOfCandidates, fmt.Errorf("no node found")
	}
	return listOfCandidates[len(listOfCandidates)-1], listOfCandidates, nil
}

func InRange(rng protocol.Range, pos protocol.Position) bool {
	if rng.Start.Line == rng.End.Line && pos.Line == rng.Start.Line {
		return rng.Start.Character <= pos.Character && pos.Character <= rng.End.Character
	}
	return rng.Start.Line <= pos.Line && pos.Line <= rng.End.Line
}

// Start is where node begins, as a protocol position.
func Start(node *sitter.Node) protocol.Position {
	point := node.StartPosition()
	return protocol.Position{Line: uint32(point.Row), Character: uint32(point.Column)}
}

// End is where node ends, as a protocol position.
func End(node *sitter.Node) protocol.Position {
	point := node.EndPosition()
	return protocol.Position{Line: uint32(point.Row), Character: uint32(point.Column)}
}
