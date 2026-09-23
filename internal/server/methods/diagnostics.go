package methods

import (
	"go.lsp.dev/protocol"

	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
)

func (methods *Methods) Diagnostics(textDocument protocol.TextDocumentItem) protocol.PublishDiagnosticsParams {
	diagnostic, _ := languageservice.DiagnosticFile(
		textDocument.URI,
		methods.Cache,
		methods.Settings,
		methods.SchemaLocation,
	)

	diagnosticParams := protocol.PublishDiagnosticsParams{
		URI:         textDocument.URI,
		Diagnostics: diagnostic,
	}

	return diagnosticParams
}
