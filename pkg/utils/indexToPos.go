package utils

import (
	"strings"

	"go.lsp.dev/protocol"
)

func IndexToPos(index int, content []byte) protocol.Position {
	line := uint32(0)
	charsOnLine := uint32(0)

	for i := 0; i < index; i++ {
		if content[i] == '\n' {
			charsOnLine = 0
			line++
		} else {
			charsOnLine++
		}
	}

	return protocol.Position{Line: line, Character: charsOnLine}
}

func PosToIndex(pos protocol.Position, content []byte) int {
	index := 0
	for i := 0; i < int(pos.Line); i++ {
		if i < int(pos.Line) {
			index = index + strings.Index(string(content[index:]), "\n") + 1
		}
	}

	index = index + int(pos.Character)
	return index
}
