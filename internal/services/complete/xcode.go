package complete

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"go.lsp.dev/protocol"
)

// xcodeBeingWritten is an xcode key with its version being written, quoted or
// not, and what is written of the version captured.
var xcodeBeingWritten = regexp.MustCompile(`^\s*xcode\s*:\s*["']?([\w.-]*)$`)

// completeMacOS completes the body of the macos at macOSLine: its xcode key,
// or the Xcode versions the catalog lists at its value. It says whether the
// cursor was in that body.
func (ch *CompletionHandler) completeMacOS(ctx context.Context, macOSLine int) bool {
	if macOSLine == -1 {
		return false
	}
	pos := ch.Params.Position
	lines := strings.Split(string(ch.Doc.Content), "\n")
	if int(pos.Line) >= len(lines) {
		return false
	}

	before := textBeforeCursor(lines[pos.Line], pos)
	if match := xcodeBeingWritten.FindStringSubmatch(before); match != nil {
		if lineAbove(lines, int(pos.Line), indentation(before)) != macOSLine {
			return false
		}
		ch.addXcodeVersions(ctx, writtenBefore(pos, match[1]))
		return true
	}

	if _, parent := ch.keyParent(); parent != macOSLine {
		return false
	}
	if !ch.stepBodyKeys(macOSLine)["xcode"] {
		ch.addCompletionItemField("xcode")
	}
	return true
}

// addXcodeVersions offers the Xcode versions the catalog lists, newest first,
// each replacing what is written of the version.
func (ch *CompletionHandler) addXcodeVersions(ctx context.Context, written protocol.Range) {
	versions := ch.Cache.Offerings(ctx, ch.Context.Client()).XcodeVersions()
	slices.SortFunc(versions, newestXcodeFirst)
	for i, version := range versions {
		ch.Items = append(ch.Items, protocol.CompletionItem{
			Label:    version,
			SortText: protocol.NewOptional(fmt.Sprintf("%04d", i)),
			TextEdit: &protocol.TextEdit{Range: written, NewText: version},
		})
	}
}

// newestXcodeFirst orders Xcode versions newest first, a version before the
// shorter one it extends, such as 16.4.0 before 16.4, and names such as
// "current" after every version.
func newestXcodeFirst(a, b string) int {
	aParts, aOK := xcodeVersionParts(a)
	bParts, bOK := xcodeVersionParts(b)
	switch {
	case aOK != bOK:
		if aOK {
			return -1
		}
		return 1
	case !aOK:
		return strings.Compare(a, b)
	}
	return -slices.Compare(aParts, bParts)
}

// xcodeVersionParts is a version's dotted numbers, and whether it is one.
func xcodeVersionParts(version string) ([]int, bool) {
	parts := []int{}
	for field := range strings.SplitSeq(version, ".") {
		n, err := strconv.Atoi(field)
		if err != nil {
			return nil, false
		}
		parts = append(parts, n)
	}
	return parts, true
}

// macOSLineIn is the line of the macos key in the body of the executor whose
// name is at executorLine, or -1.
func macOSLineIn(lines []string, executorLine int) int {
	if executorLine < 0 || executorLine >= len(lines) {
		return -1
	}
	indent := indentation(lines[executorLine])
	for line := executorLine + 1; line < len(lines); line++ {
		text := lines[line]
		if strings.TrimSpace(text) == "" {
			continue
		}
		if indentation(text) <= indent {
			break
		}
		if strings.TrimSpace(text) == "macos:" && parentLine(lines, line) == executorLine {
			return line
		}
	}
	return -1
}
