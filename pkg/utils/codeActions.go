package utils

import (
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
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
