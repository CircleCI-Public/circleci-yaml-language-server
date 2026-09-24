package methods

import (
	"bytes"
	"context"
	"path"
	"strings"

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

func (methods *Methods) DidOpen(_ context.Context, params *protocol.DidOpenTextDocumentParams) error {
	methods.setChangeInFileCache(params.TextDocument)
	methods.parsingMethods(params.TextDocument)
	methods.updateOrbFile([]byte(params.TextDocument.Text), params.TextDocument.URI)
	go (func() {
		methods.notificationMethods(params.TextDocument)
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
			go methods.notificationMethods(file.TextDocument)
		}
	})
}

func (methods *Methods) DidChange(_ context.Context, params *protocol.DidChangeTextDocumentParams) error {
	newText := methods.applyIncrementalChanges(params.TextDocument.URI, params.ContentChanges)
	textDocument := protocol.TextDocumentItem{
		URI:     params.TextDocument.URI,
		Text:    newText,
		Version: params.TextDocument.Version,
	}
	methods.setChangeInFileCache(textDocument)
	methods.updateOrbFile([]byte(newText), params.TextDocument.URI)

	methods.debounceEdit(func() {
		methods.parsingMethods(textDocument)
		go methods.notificationMethods(textDocument)
	})
	return nil
}

func (methods *Methods) DidClose(_ context.Context, params *protocol.DidCloseTextDocumentParams) error {
	// removed due to a bug in remote orbs
	isOrb, _ := methods.isOrb(params.TextDocument.URI)
	if isOrb {
		methods.Cache.FileCache.RemoveFile(params.TextDocument.URI)
		defer methods.publishDiagnostics(protocol.PublishDiagnosticsParams{
			URI:         params.TextDocument.URI,
			Diagnostics: []protocol.Diagnostic{},
		})
	}
	return nil
}

func (methods *Methods) notificationMethods(textDocument protocol.TextDocumentItem) {
	isOrb, _ := methods.isOrb(textDocument.URI)
	if methods.Settings().Api.Token != "" && !isOrb {
		methods.getAllEnvVariables(textDocument)
	}

	diagnostics := methods.Diagnostics(textDocument)

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

func (methods *Methods) parsingMethods(textDocument protocol.TextDocumentItem) {
	parsedFile, err := parser2.ParseFromUriWithCache(textDocument.URI, methods.Cache, methods.Settings())

	if err != nil {
		return
	}
	defer parsedFile.Close()

	parser2.ParseRemoteOrbs(parsedFile.Orbs, methods.Cache, methods.Settings())
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

func (methods *Methods) updateOrbFile(content []byte, uri uri.URI) {
	isOrb, orbId := methods.isOrb(uri)
	if isOrb {
		parsedOrbSource, err := parser2.ParseFromContent([]byte(content), methods.Settings(), uri, protocol.Position{})
		if err == nil {
			methods.Cache.OrbCache.UpdateOrbParsedAttributes(orbId, parsedOrbSource.ToOrbParsedAttributes())
			parsedOrbSource.Close()
		}
	}
}

func (methods *Methods) isOrb(uri uri.URI) (bool, string) {
	namespace := path.Base((path.Dir(uri.FsPath())))
	orb := path.Base(uri.FsPath())
	orbId := strings.TrimRight(path.Join(namespace, orb), ".yml")

	isOrb := methods.Cache.OrbCache.HasOrb(orbId)

	return isOrb, orbId
}
