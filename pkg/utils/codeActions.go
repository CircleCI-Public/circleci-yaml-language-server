package utils

import (
	"github.com/segmentio/encoding/json"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"strings"
)

func CreateCodeActionTextEdit(title string, textDocumentUri uri.URI, textEdits []protocol.TextEdit, isPreferred bool) protocol.CodeAction {
	return protocol.CodeAction{
		Title: title,
		Kind:  "quickfix",
		Edit: &protocol.WorkspaceEdit{
			Changes: map[uri.URI][]protocol.TextEdit{
				textDocumentUri: textEdits,
			},
		},
		IsPreferred: isPreferred,
	}
}

func AppendSuppressionCodeActions(docURI protocol.URI, diagnostics []protocol.Diagnostic, docContent []byte) ([]protocol.Diagnostic, error) {
	newDiagnostics := []protocol.Diagnostic{}

	docLines := strings.Split(string(docContent), "\n")

	for _, diagnostic := range diagnostics {
		// Extract indentation from the line where diagnostic occurs
		indent := ""
		if int(diagnostic.Range.Start.Line) < len(docLines) {
			lineText := docLines[diagnostic.Range.Start.Line]
			for _, ch := range lineText {
				if ch == ' ' || ch == '\t' {
					indent += string(ch)
				} else {
					break
				}
			}
		}

		// Suggest to ignore next line first as it applies to both single and multi-line diagnostics.
		ignoreActions := []protocol.CodeAction{
			CreateCodeActionTextEdit(
				"Ignore this line",
				docURI,
				[]protocol.TextEdit{
					{
						NewText: indent + "# cci-ignore-next-line\n",
						Range: protocol.Range{
							Start: protocol.Position{
								Character: 0,
								Line:      diagnostic.Range.Start.Line,
							},
							End: protocol.Position{
								Character: 0,
								Line:      diagnostic.Range.Start.Line,
							},
						},
					},
				},
				false, // not preferred
			),
		}

		if diagnostic.Range.Start.Line == diagnostic.Range.End.Line {
			// Single-line diagnostic, offer to put an inline comment
			ignoreActions = append(ignoreActions, CreateCodeActionTextEdit(
				"Ignore this line (inline)",
				docURI,
				[]protocol.TextEdit{
					{
						NewText: " # cci-ignore",
						Range: protocol.Range{
							Start: protocol.Position{
								Character: diagnostic.Range.End.Character + 1,
								Line:      diagnostic.Range.Start.Line,
							},
							End: protocol.Position{
								Character: diagnostic.Range.End.Character + 14,
								Line:      diagnostic.Range.Start.Line,
							},
						},
					},
				},
				false, // not preferred
			))
		} else {
			// Otherwise it's a multi-line diagnostic. Offer to ignore a range.
			// NOTE: This doesn't handle if there are separate diagnostics on multiple lines in a row.
			ignoreActions = append(ignoreActions, CreateCodeActionTextEdit(
				"Ignore this range",
				docURI,
				[]protocol.TextEdit{
					{
						NewText: indent + "# cci-ignore-start\n",
						Range: protocol.Range{
							Start: protocol.Position{
								Character: 0,
								Line:      diagnostic.Range.Start.Line,
							},
							End: protocol.Position{
								Character: 0,
								Line:      diagnostic.Range.Start.Line,
							},
						},
					},
					{
						NewText: indent + "# cci-ignore-end\n",
						Range: protocol.Range{
							Start: protocol.Position{
								Character: 0,
								Line:      diagnostic.Range.End.Line + 1,
							},
							End: protocol.Position{
								Character: 0,
								Line:      diagnostic.Range.End.Line + 1,
							},
						},
					},
				},
				false, // not preferred
			))
		}

		// Suggest to ignore the entire file last as we don't want to encourage this.
		ignoreActions = append(ignoreActions,
			CreateCodeActionTextEdit(
				"Ignore all errors in this file",
				docURI,
				[]protocol.TextEdit{
					{
						NewText: "# cci-ignore-file\n",
						Range: protocol.Range{
							Start: protocol.Position{
								Character: 0,
								Line:      0,
							},
							End: protocol.Position{
								Character: 0,
								Line:      0,
							},
						},
					},
				},
				false, // not preferred
			),
		)

		// Append to existing code actions if there are any.
		existingActions := []protocol.CodeAction{}
		if diagnostic.Data != nil {
			// Unmarshal existing code actions
			str, err := json.Marshal(diagnostic.Data)
			if err != nil {
				return nil, err
			}

			err = json.Unmarshal(str, &existingActions)
			if err != nil {
				return nil, err
			}
		}

		diagnostic.Data = append(existingActions, ignoreActions...)
		newDiagnostics = append(newDiagnostics, diagnostic)
	}

	return newDiagnostics, nil
}
