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

	indent := len(before) - len(strings.TrimLeft(before, " "))
	for parent := int(pos.Line) - 1; parent >= 0; parent-- {
		text := lines[parent]
		if strings.TrimSpace(text) != "" && len(text)-len(strings.TrimLeft(text, " ")) < indent {
			return lines, parent
		}
	}

	return lines, -1
}
