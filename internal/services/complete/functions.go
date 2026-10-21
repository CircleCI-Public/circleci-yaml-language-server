package complete

import (
	"context"
	"regexp"
	"strings"

	"go.lsp.dev/protocol"
	"golang.org/x/mod/module"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
)

// functionNameBeingWritten is a function's declaration with its name being
// written, before any @, and what is written of it captured.
var functionNameBeingWritten = regexp.MustCompile(`^\s*[\w-]+\s*:\s*([^\s@]*)$`)

// functionVersionBeingWritten is a function's declaration with its version
// being written after the @, and the function's path and what is written of
// the version captured.
var functionVersionBeingWritten = regexp.MustCompile(`^\s*[\w-]+\s*:\s*([^\s@]+)@([\w.-]*)$`)

// completeFunctionDeclaration offers, at a declared function, the published
// functions at its name, or the function's versions at its version.
func (ch *CompletionHandler) completeFunctionDeclaration(ctx context.Context) {
	if !ch.completeFunctionName(ctx) {
		ch.completeFunctionVersion(ctx)
	}
}

// completeFunctionName offers each published function, at its latest
// version, at a declared function's name. It says whether the cursor is at
// one.
func (ch *CompletionHandler) completeFunctionName(ctx context.Context) bool {
	pos := ch.Params.Position
	lines := strings.Split(string(ch.Doc.Content), "\n")
	if int(pos.Line) >= len(lines) {
		return false
	}

	line := lines[pos.Line]
	before := line[:min(int(pos.Character), len(line))]
	match := functionNameBeingWritten.FindStringSubmatch(before)
	if match == nil {
		return false
	}

	functions, err := ch.Cache.Functions.All(ctx, ch.Context.Client())
	if err != nil {
		return true
	}

	written := writtenBefore(pos, match[1])
	for _, function := range functions {
		reference := function.Name
		if function.LatestVersion != "" && !module.IsPseudoVersion(function.LatestVersion) {
			reference += "@" + function.LatestVersion
		}
		item := protocol.CompletionItem{
			Label:    reference,
			Kind:     protocol.CompletionItemKindModule,
			TextEdit: &protocol.TextEdit{Range: written, NewText: reference},
		}
		if function.Description != "" {
			item.Documentation = &protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: function.Description}
		}
		ch.Items = append(ch.Items, item)
	}
	return true
}

// writtenBefore is the range of what is written just before pos. A function
// item replaces what is written of its name or version rather than leaving a
// client to find it: VS Code and IntelliJ each find the word at the cursor in
// their own way, splitting it at a / . or @ or not, and IntelliJ drops every
// item that doesn't start with the word it found.
func writtenBefore(pos protocol.Position, written string) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{Line: pos.Line, Character: pos.Character - uint32(len(written))},
		End:   pos,
	}
}

// completeFunctionVersion offers the published versions of a declared
// function, at its version.
func (ch *CompletionHandler) completeFunctionVersion(ctx context.Context) {
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

	published, err := ch.Cache.Functions.Function(ctx, ch.Context.Client(), match[1])
	if err != nil || published == nil {
		return
	}
	written := writtenBefore(pos, match[2])
	for _, version := range published.Versions {
		if module.IsPseudoVersion(version.Version) {
			continue
		}
		ch.Items = append(ch.Items, protocol.CompletionItem{
			Label:    version.Version,
			TextEdit: &protocol.TextEdit{Range: written, NewText: version.Version},
		})
	}
}

// completeFunctionFlags offers the flags a function step doesn't pass yet,
// at a key of its `with`, or a boolean flag's values or a flag's default, at
// its value. It says whether the cursor is in a function step's `with`.
func (ch *CompletionHandler) completeFunctionFlags(ctx context.Context) bool {
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

	_, descriptor, err := ch.Doc.LookUpFunction(ctx, function, ch.Cache)
	if err != nil || descriptor == nil {
		return true
	}
	flags, _ := descriptor.StepFlags(command)

	if atValue {
		for _, flag := range flags {
			if flag.Name != key {
				continue
			}
			if flag.Type == "bool" {
				ch.addCompletionItem("true")
				ch.addCompletionItem("false")
			} else if value, ok := flag.DefaultValue(); ok {
				ch.addCompletionItemWithDetail(value, "default", "")
			}
		}
		return true
	}

	present := ch.stepBodyKeys(withLine)
	for _, flag := range flags {
		if !present[flag.Name] {
			ch.addFunctionFlag(flag)
		}
	}
	return true
}

func (ch *CompletionHandler) addFunctionFlag(flag circleci.FunctionFlag) {
	detail := flag.Type
	if value, ok := flag.DefaultValue(); ok {
		detail += ", default " + value
	}
	item := protocol.CompletionItem{
		Label:      flag.Name,
		Kind:       protocol.CompletionItemKindProperty,
		Detail:     unlessEmpty(detail),
		InsertText: protocol.NewOptional(flag.Name + ": "),
	}
	if flag.Description != "" {
		item.Documentation = &protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: flag.Description}
	}
	ch.Items = append(ch.Items, item)
}
