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

// completeFunctionFlags offers the flags a function step doesn't pass yet,
// at a key of its `with`, or a boolean flag's values, at its value. It says
// whether the cursor is in a function step's `with`.
func (ch *CompletionHandler) completeFunctionFlags() bool {
	key, lines, withLine := ch.valueAt()
	atValue := withLine != -1
	if !atValue {
		lines, withLine = ch.keyParent()
	}
	if withLine == -1 || strings.TrimSpace(lines[withLine]) != "with:" {
		return false
	}

	stepLine := parentLine(lines, withLine)
	if stepLine == -1 {
		return false
	}
	match := stepWithBody.FindStringSubmatch(lines[stepLine])
	if match == nil {
		return false
	}
	function, command, ok := ch.Doc.FunctionForStep(match[2])
	if !ok {
		return false
	}

	_, descriptor, err := ch.Doc.LookUpFunction(function, ch.Cache)
	if err != nil || descriptor == nil {
		return true
	}
	flags := descriptor.Flags
	if command != "" {
		flags = descriptor.Commands[command].Flags
	}

	if atValue {
		for _, flag := range flags {
			if flag.Name == key && flag.Type == "bool" {
				ch.addCompletionItem("true")
				ch.addCompletionItem("false")
			}
		}
		return true
	}

	present := ch.stepBodyKeys(withLine)
	for _, flag := range flags {
		if !present[flag.Name] {
			ch.addCompletionItemFieldWithCustomText(flag.Name, "", ": ", flag.Description, "")
		}
	}
	return true
}
