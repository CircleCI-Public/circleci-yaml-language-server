package complete

import (
	"regexp"
	"strings"
)

// functionVersionBeingWritten is a function's declaration with its version
// being written after the @, and the function's path captured.
var functionVersionBeingWritten = regexp.MustCompile(`^\s*[\w-]+\s*:\s*([^\s@]+)@[\w.-]*$`)

// completeFunctionVersion offers the published versions of a declared
// function, at its version.
func (ch *CompletionHandler) completeFunctionVersion() {
	pos := ch.Params.Position
	lines := strings.Split(string(ch.Doc.Content), "\n")
	if int(pos.Line) >= len(lines) {
		return
	}

	line := lines[pos.Line]
	match := functionVersionBeingWritten.FindStringSubmatch(line[:min(int(pos.Character), len(line))])
	if match == nil {
		return
	}

	published, err := ch.Cache.Functions.Function(ch.Context.V3Client(), match[1])
	if err != nil || published == nil {
		return
	}
	for _, version := range published.Versions {
		ch.addCompletionItem(version.Version)
	}
}
