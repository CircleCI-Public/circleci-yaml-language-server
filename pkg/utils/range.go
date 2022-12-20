package utils

import (
	"strings"

	"go.lsp.dev/protocol"
)

func OffsetRange(rng *protocol.Range, offset protocol.Position) {
	rng.Start.Line += offset.Line
	rng.Start.Character += offset.Character
	rng.End.Line += offset.Line
	rng.End.Character += offset.Character
}

// Return the exact range of the text on a line
//
// For example, with a yaml:
//
// content := `
//
//	some-key:
//	  property: 5
//
// `
//
//	LineContentRange(2, []byte(content)) ==> {
//		Start: {
//		 	Line: 2
//			Character: 2
//		},
//		End: {
//			Line: 2,
//			Character: 12
//	  }
//	}
func LineContentRange(lineIndex int, content []byte) protocol.Range {
	ranges := AllLineContentRange([]int{lineIndex}, content)

	return ranges[0]
}

// Return the exact range of the text on a all given lines.
// See LineContentRange for more information
func AllLineContentRange(lineIndexes []int, content []byte) []protocol.Range {
	str := string(content)

	allLines := strings.Split(str, "\n")

	ranges := []protocol.Range{}

	for _, lineIndex := range lineIndexes {
		line := allLines[lineIndex]

		trim := strings.TrimSpace(line)

		index := strings.Index(line, trim)

		contentRange := protocol.Range{
			Start: protocol.Position{
				Line:      uint32(lineIndex),
				Character: uint32(index),
			},

			End: protocol.Position{
				Line:      uint32(lineIndex),
				Character: uint32(index + len(line)),
			},
		}

		ranges = append(ranges, contentRange)
	}

	return ranges
}

// Return true if two positions are identical
// Two positions are identical if they ave the same line and character.
func ArePositionEqual(a protocol.Position, b protocol.Position) bool {
	return ComparePosition(a, b) == 0
}

// Return true if two ranges are identical
// Two ranges are identical if they have the same start and end position
func AreRangeEqual(a protocol.Range, b protocol.Range) bool {
	if !ArePositionEqual(a.Start, b.Start) {
		return false
	}

	return ArePositionEqual(b.End, b.End)
}

func IsDefaultRange(rng protocol.Range) bool {
	// A default range is a set of default position
	// which is a set of numbers which defaults to 0 in Go

	return (rng.Start.Character + rng.Start.Line + rng.End.Character + rng.End.Line) == 0
}

// Compare two positions.
// Return 0 if the two position are the same
// Return 1 if a is before b
// Return -1 if a is after b
func ComparePosition(a protocol.Position, b protocol.Position) int {
	if a.Line == b.Line {
		diff := b.Character - a.Character

		if diff == 0 {
			return 0
		}

		if diff > 0 {
			return 1
		}

		return -1
	}

	diff := b.Line - a.Line

	if diff == 0 {
		return 0
	}

	if diff > 0 {
		return 1
	}

	return -1
}
