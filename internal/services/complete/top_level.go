package complete

import (
	"regexp"
	"strings"
)

// topLevelKeys are the keys a config takes at its top level.
var topLevelKeys = []string{
	"version", "setup", "orbs", "functions", "parameters", "executors",
	"commands", "jobs", "job-groups", "workflows",
}

var (
	topLevelKeyBeingWritten = regexp.MustCompile(`^[\w-]*$`)
	versionBeingWritten     = regexp.MustCompile(`^version\s*:\s*[\d.]*$`)
	topLevelKey             = regexp.MustCompile(`^([\w-]+)\s*:`)
)

// completeTopLevel offers the top-level keys a config doesn't have yet, or
// the version of the config format, when the cursor is at one, and says
// whether it is.
func (ch *CompletionHandler) completeTopLevel() bool {
	pos := ch.Params.Position
	lines := strings.Split(string(ch.Doc.Content), "\n")
	if int(pos.Line) >= len(lines) || int(pos.Character) > len(lines[pos.Line]) {
		return false
	}

	before := lines[pos.Line][:pos.Character]
	switch {
	case versionBeingWritten.MatchString(before):
		ch.addCompletionItem("2.1")
		return true
	case !topLevelKeyBeingWritten.MatchString(before):
		return false
	}

	present := map[string]bool{}
	for i, text := range lines {
		if match := topLevelKey.FindStringSubmatch(text); match != nil && i != int(pos.Line) {
			present[match[1]] = true
		}
	}
	for _, key := range topLevelKeys {
		if !present[key] {
			ch.addCompletionItemField(key)
		}
	}
	return true
}
