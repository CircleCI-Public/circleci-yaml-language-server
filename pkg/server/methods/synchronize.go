package methods

import (
	"bytes"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	lsp "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/services"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"github.com/bep/debounce"
	"github.com/segmentio/encoding/json"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

func (methods *Methods) DidOpen(reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	params := protocol.DidOpenTextDocumentParams{}
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(methods.Ctx, nil, fmt.Errorf("%s: %w", jsonrpc2.ErrParse, err))
	}

	methods.Cache.FileCache.SetFile(&params.TextDocument)
	methods.parsingMethods(params.TextDocument)
	methods.updateOrbFile([]byte(params.TextDocument.Text), params.TextDocument.URI)
	go (func() {
		methods.notificationMethods(methods.Cache.FileCache, params.TextDocument)
		methods.SendTelemetryEvent(TelemetryEvent{
			Event:  "DidOpen",
			Action: "finished",
			Properties: DidOpenFinishedProperties{
				Filename: params.TextDocument.URI.Filename(),
			},
		})
	})()

	return reply(methods.Ctx, nil, nil)
}

var debounceInnerChange = debounce.New(1000 * time.Millisecond)

func (methods *Methods) DidChange(reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	params := protocol.DidChangeTextDocumentParams{}
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(methods.Ctx, nil, fmt.Errorf("%s: %w", jsonrpc2.ErrParse, err))
	}
	newText := methods.applyIncrementalChanges(params.TextDocument.URI, params.ContentChanges)
	textDocument := protocol.TextDocumentItem{
		URI:     params.TextDocument.URI,
		Text:    newText,
		Version: params.TextDocument.Version,
	}
	methods.Cache.FileCache.SetFile(&textDocument)
	methods.updateOrbFile([]byte(newText), params.TextDocument.URI)

	debounceInnerChange(func() {
		methods.parsingMethods(textDocument)
		go methods.notificationMethods(methods.Cache.FileCache, textDocument)
	})
	return reply(methods.Ctx, nil, nil)
}

func (methods *Methods) DidClose(reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	params := protocol.DidCloseTextDocumentParams{}
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(methods.Ctx, nil, fmt.Errorf("%s: %w", jsonrpc2.ErrParse, err))
	}

	// removed due to a bug in remote orbs
	isOrb, _ := methods.isOrb(params.TextDocument.URI)
	if isOrb {
		methods.Cache.FileCache.RemoveFile(params.TextDocument.URI)
		defer methods.Conn.Notify(
			methods.Ctx,
			protocol.MethodTextDocumentPublishDiagnostics,
			protocol.PublishDiagnosticsParams{
				URI:         params.TextDocument.URI,
				Diagnostics: []protocol.Diagnostic{},
			},
		)
	}

	return reply(methods.Ctx, nil, nil)
}

func (methods *Methods) notificationMethods(cache utils.FileCache, textDocument protocol.TextDocumentItem) {
	diagnostic, _ := lsp.DiagnosticFile(
		textDocument.URI,
		methods.Cache,
		methods.LsContext,
		methods.SchemaLocation,
	)

	// TODO: Handle error

	diagnosticParams := protocol.PublishDiagnosticsParams{
		URI:         textDocument.URI,
		Diagnostics: diagnostic,
	}

	original := cache.GetFile(textDocument.URI)

	// Compare the version
	// To avoid notifying based on an older version document
	if original != nil && original.Version == textDocument.Version {
		err := methods.Conn.Notify(
			methods.Ctx,
			protocol.MethodTextDocumentPublishDiagnostics,
			diagnosticParams,
		)

		if err != nil {
			// TODO: Handle error
		}
	}

}

func (methods *Methods) parsingMethods(textDocument protocol.TextDocumentItem) {
	parsedFile, err := yamlparser.ParseFromUriWithCache(textDocument.URI, methods.Cache, methods.LsContext)

	if err != nil {
		return
	}

	yamlparser.ParseRemoteOrbs(parsedFile.Orbs, methods.Cache, methods.LsContext)
}

func (methods *Methods) applyIncrementalChanges(uri protocol.URI, changes []protocol.TextDocumentContentChangeEvent) string {
	file := methods.Cache.FileCache.GetFile(uri)
	content := []byte(file.Text)

	for _, change := range changes {
		start, end := utils.PosToIndex(change.Range.Start, content), utils.PosToIndex(change.Range.End, content)

		var buf bytes.Buffer
		buf.Write(content[:start])
		buf.Write([]byte(change.Text))
		buf.Write(content[end:])
		content = buf.Bytes()
	}

	return string(content)
}

func (methods *Methods) updateOrbFile(content []byte, uri protocol.URI) {
	isOrb, orbId := methods.isOrb(uri)
	if isOrb {
		parsedOrbSource, err := parser.ParseFromContent([]byte(content), methods.LsContext, uri)
		if err == nil {
			methods.Cache.OrbCache.UpdateOrbParsedAttributes(orbId, parsedOrbSource.ToOrbParsedAttributes())
		}
	}
}

func (methods *Methods) isOrb(uri protocol.URI) (bool, string) {
	namespace := path.Base((path.Dir(uri.Filename())))
	orb := path.Base(uri.Filename())
	orbId := strings.TrimRight(path.Join(namespace, orb), ".yml")

	isOrb := methods.Cache.OrbCache.HasOrb(orbId)

	return isOrb, orbId
}
