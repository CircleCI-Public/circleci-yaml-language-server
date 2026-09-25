package complete

import (
	"regexp"
	"strings"
)

var keyBeingWritten = regexp.MustCompile(`^\s*[\w-]*$`)

// keyParent is the line of the mapping key or list item whose body the
// cursor is at a key of, with the document's lines, or -1 when the cursor
// isn't where a key is written. Like stepAt, it reads the indentation, since
// a node's range doesn't reach a blank line after its body.
func (ch *CompletionHandler) keyParent() ([]string, int) {
	pos := ch.Params.Position
	lines := strings.Split(string(ch.Doc.Content), "\n")
	if int(pos.Line) >= len(lines) {
		return lines, -1
	}

	line := lines[pos.Line]
	before := line[:min(int(pos.Character), len(line))]
	if !keyBeingWritten.MatchString(before) {
		return lines, -1
	}

	return lines, lineAbove(lines, int(pos.Line), indentation(before))
}

// parentLine is the line of the key or list item a line is in the body of,
// or -1 when it is at the top level.
func parentLine(lines []string, line int) int {
	return lineAbove(lines, line, indentation(lines[line]))
}

// lineAbove is the nearest line above one that isn't blank and is indented
// less than indent, or -1.
func lineAbove(lines []string, line, indent int) int {
	for above := line - 1; above >= 0; above-- {
		if strings.TrimSpace(lines[above]) != "" && indentation(lines[above]) < indent {
			return above
		}
	}
	return -1
}

func indentation(text string) int {
	return len(text) - len(strings.TrimLeft(text, " "))
}
