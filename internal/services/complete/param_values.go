package complete

import (
	"regexp"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
)

var valueBeingWritten = regexp.MustCompile(`^(\s*)([A-Za-z_][\w-]*)\s*:\s*[\w./:-]*$`)

// valueAt is the key whose value the cursor is at, on the key's own line,
// with the line of the mapping key or list item whose body the key is in.
// The line is -1 when the cursor isn't at such a value.
func (ch *CompletionHandler) valueAt() (string, []string, int) {
	pos := ch.Params.Position
	lines := strings.Split(string(ch.Doc.Content), "\n")
	if int(pos.Line) >= len(lines) {
		return "", lines, -1
	}

	line := lines[pos.Line]
	match := valueBeingWritten.FindStringSubmatch(line[:min(int(pos.Character), len(line))])
	if match == nil {
		return "", lines, -1
	}

	parent := lineAbove(lines, int(pos.Line), len(match[1]))
	if parent == -1 {
		return "", lines, -1
	}
	return match[2], lines, parent
}

// addParameterValues offers the values an enum or boolean parameter takes.
func (ch *CompletionHandler) addParameterValues(param ast.Parameter) {
	switch param := param.(type) {
	case ast.EnumParameter:
		for _, value := range param.Enum {
			ch.addCompletionItem(value)
		}
	case ast.BooleanParameter:
		ch.addCompletionItem("true")
		ch.addCompletionItem("false")
	}
}
