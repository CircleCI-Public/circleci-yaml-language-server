package complete

import (
	"regexp"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
)

var (
	// contextBeingWritten is a context: key with its value, alone or in a
	// flow list, being written.
	contextBeingWritten  = regexp.MustCompile(`^\s*context\s*:\s*(\[(?:\s*[\w./-]+\s*,)*)?\s*[\w./-]*$`)
	listItemBeingWritten = regexp.MustCompile(`^\s*-\s*[\w./-]*$`)
	contextKey           = regexp.MustCompile(`^\s*context\s*:\s*$`)
)

// completeContextName offers the names of the organization's contexts, when
// the cursor is at a context of a job invocation, and says whether it is.
func (ch *CompletionHandler) completeContextName(invocations []ast.JobInvocation) bool {
	pos := ch.Params.Position
	lines := strings.Split(string(ch.Doc.Content), "\n")
	if int(pos.Line) >= len(lines) {
		return false
	}

	line := lines[pos.Line]
	before := line[:min(int(pos.Character), len(line))]
	contextLine := int(pos.Line)
	if listItemBeingWritten.MatchString(before) {
		contextLine = parentLine(lines, contextLine)
		if contextLine == -1 || !contextKey.MatchString(lines[contextLine]) {
			return false
		}
	} else if !contextBeingWritten.MatchString(before) {
		return false
	}

	invocationLine := parentLine(lines, contextLine)
	if invocationLine == -1 || invocationNamedOn(invocationLine, invocations) == nil {
		return false
	}

	if cachedFile := ch.Cache.FileCache.GetFile(ch.Doc.URI); cachedFile != nil {
		for _, name := range ch.Cache.ContextCache.ContextNames(cachedFile.Project.OrganizationId) {
			ch.addCompletionItem(name)
		}
	}
	return true
}
