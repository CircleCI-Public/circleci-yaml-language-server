package complete

import (
	"regexp"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser/validate"
)

var (
	// statusBeingWritten is a required job with the status it's required
	// to have being written, alone or in a list, whose start the group
	// captures.
	statusBeingWritten = regexp.MustCompile(`^\s*-\s+[\w/.-]+\s*:\s*(\[(?:\s*[\w-]+\s*,)*)?\s*[\w-]*$`)
	requiresKey        = regexp.MustCompile(`^\s*requires\s*:\s*$`)
)

// completeRequiredStatus offers the statuses a required job can be required
// to have, when the cursor is at one, and says whether it is. A list of
// statuses can't include terminal, which stands for all of them.
func (ch *CompletionHandler) completeRequiredStatus() bool {
	pos := ch.Params.Position
	lines := strings.Split(string(ch.Doc.Content), "\n")
	if int(pos.Line) >= len(lines) {
		return false
	}

	line := lines[pos.Line]
	match := statusBeingWritten.FindStringSubmatch(line[:min(int(pos.Character), len(line))])
	if match == nil {
		return false
	}
	if parent := parentLine(lines, int(pos.Line)); parent == -1 || !requiresKey.MatchString(lines[parent]) {
		return false
	}

	for _, status := range validate.TerminalJobStatuses {
		ch.addCompletionItem(status)
	}
	if match[1] == "" {
		ch.addCompletionItem("terminal")
	}
	return true
}
