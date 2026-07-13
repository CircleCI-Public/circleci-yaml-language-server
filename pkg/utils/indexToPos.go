package utils

import (
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
