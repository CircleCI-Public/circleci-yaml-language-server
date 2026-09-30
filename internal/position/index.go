package position

import (
	"go.lsp.dev/protocol"
)

func FromIndex(index int, content []byte) protocol.Position {
	return Advance(protocol.Position{}, content[:index])
}

// Advance is the position just past text, which starts at from. Given where
// a node starts, it finds a place in the node's text without reading all the
// content before it, as FromIndex does.
func Advance(from protocol.Position, text []byte) protocol.Position {
	for _, b := range text {
		if b == '\n' {
			from.Line++
			from.Character = 0
		} else {
			from.Character++
		}
	}
	return from
}

func ToIndex(pos protocol.Position, content []byte) int {
	if len(content) == 0 {
		return 0
	}

	idx := 0
	line := uint32(0)
	for idx < len(content) && line < pos.Line {
		if content[idx] == '\n' {
			line++
		}
		idx++
	}

	target := idx + int(pos.Character)
	if target > len(content) {
		return len(content)
	}

	return target
}

// Lines finds indexes in a content many times, without reading it up to each
// position as ToIndex does.
type Lines struct {
	starts []int
	length int
}

// NewLines reads where each of content's lines starts.
func NewLines(content []byte) Lines {
	starts := []int{0}
	for i, b := range content {
		if b == '\n' {
			starts = append(starts, i+1)
		}
	}
	return Lines{starts: starts, length: len(content)}
}

// ToIndex is ToIndex for the content lines was made from.
func (lines Lines) ToIndex(pos protocol.Position) int {
	if int(pos.Line) >= len(lines.starts) {
		return lines.length
	}
	return min(lines.starts[pos.Line]+int(pos.Character), lines.length)
}
