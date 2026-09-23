package methods

import (
	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/services"
	"go.lsp.dev/protocol"
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
