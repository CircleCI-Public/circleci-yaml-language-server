package utils

import (
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

func NodeAtPos(rootNode *sitter.Node, pos protocol.Position) (*sitter.Node, []*sitter.Node, error) {
	iterator := sitter.NewIterator(rootNode, sitter.DFSMode)
	listOfCandidates := make([]*sitter.Node, 0)

	iterator.ForEach(func(node *sitter.Node) error {
		rng := protocol.Range{
			Start: protocol.Position{
				Line:      node.StartPoint().Row,
				Character: node.StartPoint().Column,
			},
			End: protocol.Position{
				Line:      node.EndPoint().Row,
				Character: node.EndPoint().Column,
			},
		}
		if PosInRange(rng, pos) {
			listOfCandidates = append(listOfCandidates, node)
		}
		return nil
	})

	if len(listOfCandidates) == 0 {
		return nil, listOfCandidates, fmt.Errorf("no node found")
	}
	return listOfCandidates[len(listOfCandidates)-1], listOfCandidates, nil
}

func PosInRange(rng protocol.Range, pos protocol.Position) bool {
	if rng.Start.Line == rng.End.Line && pos.Line == rng.Start.Line {
		return rng.Start.Character <= pos.Character && pos.Character <= rng.End.Character
	}
	return rng.Start.Line <= pos.Line && pos.Line <= rng.End.Line
}
