package ir

import (
	"bytes"
	"unicode/utf8"
)

// Loc is a position in a source file. Line and Col are 1-indexed, and Col
// counts code points.
type Loc struct {
	Source string `json:"source,omitempty"`
	Line   int    `json:"line"`
	Col    int    `json:"col"`
}

// Frame is one step of the expansion that produced a value, such as a job or
// command call.
type Frame struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	Loc  Loc    `json:"loc"`
}

// LocOf converts a 0-indexed line and UTF-8 byte column within content to a
// Loc. An inline orb's positions are in its parent's coordinates, so pass the
// parent's content.
func LocOf(source string, content []byte, line, byteCol int) Loc {
	loc := Loc{Source: source, Line: line + 1, Col: byteCol + 1}
	for range line {
		i := bytes.IndexByte(content, '\n')
		if i < 0 {
			return loc
		}
		content = content[i+1:]
	}
	if i := bytes.IndexByte(content, '\n'); i >= 0 {
		content = content[:i]
	}
	n := min(byteCol, len(content))
	loc.Col = utf8.RuneCount(content[:n]) + byteCol - n + 1
	return loc
}
