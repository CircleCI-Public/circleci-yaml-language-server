package methods

import (
	"bytes"
	"context"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	parser2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func (methods *Methods) setChangeInFileCache(textDocument protocol.TextDocumentItem) {
	if cachedFile := methods.Cache.FileCache.GetFile(textDocument.URI); cachedFile != nil {
		methods.Cache.FileCache.UpdateTextDocument(textDocument.URI, textDocument)
	} else {
		methods.Cache.FileCache.SetFile(cache.File{
			TextDocument: textDocument,
		})
	}
}

func (methods *Methods) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) error {
	methods.setChangeInFileCache(params.TextDocument)
	methods.parsingMethods(ctx, params.TextDocument)
	methods.updateOrbFile(ctx, []byte(params.TextDocument.Text), params.TextDocument.URI)
	// What runs on after the notification is handled is the session's, not
	// the notification's.
	go (func() {
		methods.notificationMethods(methods.Ctx, params.TextDocument)
		methods.SetResourceClassOfFile(*params)
		methods.SendTelemetryEvent(TelemetryEvent{
			Action: "opened_file",
			Properties: map[string]interface{}{
				"filename": params.TextDocument.URI.FsPath(),
			},
			TriggerType: "frontend_interaction",
			Object:      "lsp",
		})
	})()
	return nil
}

func (methods *Methods) updateAllCachedFiles() {
	methods.debounceRevalidation(func() {
		files := methods.Cache.FileCache.GetFiles()

		for _, file := range files {
			go methods.notificationMethods(methods.Ctx, file.TextDocument)
		}
	})
}

func (methods *Methods) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) error {
	newText := methods.applyIncrementalChanges(params.TextDocument.URI, params.ContentChanges)
	textDocument := protocol.TextDocumentItem{
		URI:     params.TextDocument.URI,
		Text:    newText,
		Version: params.TextDocument.Version,
	}
	methods.setChangeInFileCache(textDocument)
	methods.updateOrbFile(ctx, []byte(newText), params.TextDocument.URI)

	methods.debounceEdit(func() {
		methods.parsingMethods(methods.Ctx, textDocument)
		go methods.refreshInlayHints()
		go methods.notificationMethods(methods.Ctx, textDocument)
	})
	return nil
}

// DidClose forgets a document, so that it is not validated again, and clears
// its diagnostics: the client shows them for as long as they were the last
// published.
func (methods *Methods) DidClose(_ context.Context, params *protocol.DidCloseTextDocumentParams) error {
	methods.Cache.ForgetFile(params.TextDocument.URI)
	methods.publishDiagnostics(protocol.PublishDiagnosticsParams{
		URI:         params.TextDocument.URI,
		Diagnostics: []protocol.Diagnostic{},
	})
	return nil
}

func (methods *Methods) notificationMethods(ctx context.Context, textDocument protocol.TextDocumentItem) {
	isOrb, _ := methods.isOrb(textDocument.URI)
	if methods.Settings().Api.Token != "" && !isOrb {
		methods.getAllEnvVariables(textDocument)
	}

	diagnostics := methods.Diagnostics(ctx, textDocument)

	original := methods.Cache.FileCache.GetFile(textDocument.URI)

	// Compare the version
	// To avoid notifying based on an older version document
	if original != nil && original.TextDocument.Version == textDocument.Version {
		methods.publishDiagnostics(diagnostics)

		methods.SendTelemetryEvent(TelemetryEvent{
			Object:      "lsp",
			TriggerType: "background_event",
			Action:      "run_diagnostics",
			Properties: map[string]interface{}{
				"filename": textDocument.URI.FsPath(),
			},
		})
	}

}

func (methods *Methods) parsingMethods(ctx context.Context, textDocument protocol.TextDocumentItem) {
	parsedFile, err := parser2.ParseFromUriWithCache(ctx, textDocument.URI, methods.Cache, methods.Settings())

	if err != nil {
		return
	}
	defer parsedFile.Close()

	parser2.ParseRemoteOrbs(ctx, parsedFile.Orbs, methods.Cache, methods.Settings())
}

func (methods *Methods) applyIncrementalChanges(uri uri.URI, changes []protocol.TextDocumentContentChangeEvent) string {
	file := methods.Cache.FileCache.GetFile(uri)
	content := []byte(file.TextDocument.Text)

	for _, change := range changes {
		switch change := change.(type) {
		case *protocol.TextDocumentContentChangePartial:
			start, end := position.ToIndex(change.Range.Start, content), position.ToIndex(change.Range.End, content)

			var buf bytes.Buffer
			buf.Write(content[:start])
			buf.Write([]byte(change.Text))
			buf.Write(content[end:])
			content = buf.Bytes()
		case *protocol.TextDocumentContentChangeWholeDocument:
			content = []byte(change.Text)
		}
	}

	return string(content)
}

func (methods *Methods) updateOrbFile(ctx context.Context, content []byte, uri uri.URI) {
	isOrb, orbId := methods.isOrb(uri)
	if isOrb {
		parsedOrbSource, err := parser2.ParseFromContent(ctx, []byte(content), methods.Settings(), uri, protocol.Position{})
		if err == nil {
			methods.Cache.OrbCache.UpdateOrbParsedAttributes(orbId, parsedOrbSource.ToOrbParsedAttributes())
			parsedOrbSource.Close()
		}
	}
}

// isOrb reports whether a document is the source of a remote orb, which the
// server wrote out for go-to-definition to open, and the reference of the orb.
func (methods *Methods) isOrb(uri uri.URI) (bool, string) {
	orbId, isOrb := methods.Cache.OrbIDOfSource(uri.FsPath())

	return isOrb, orbId
}
