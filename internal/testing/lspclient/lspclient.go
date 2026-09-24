// Package lspclient is a language server client for acceptance tests.
//
// It wraps jsonrpc2 with the handful of requests the tests make, and collects
// what the server sends the other way: diagnostics are published as
// notifications rather than returned from a call, so a test that wants them
// has to be listening before it opens a document.
//
// Every wait is bounded. A server that never answers should fail the test that
// asked, rather than hang until the test binary's own timeout takes the whole
// package down with no indication of which case was stuck.
package lspclient

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/lspcodec"
)

// DefaultTimeout bounds a request, and a wait for diagnostics.
const DefaultTimeout = 20 * time.Second

// pollInterval is how often a wait re-checks what has arrived.
const pollInterval = 10 * time.Millisecond

// Client is a language server client over one connection.
type Client struct {
	ctx     context.Context
	conn    jsonrpc2.Conn
	timeout time.Duration

	mu          sync.Mutex
	published   map[uri.URI]publication
	consumed    map[uri.URI]int
	telemetry   []map[string]any
	logMessages []string
}

// publication is the diagnostics last published for a document, and how many
// times the server has published for it.
type publication struct {
	count int
	items []protocol.Diagnostic
}

// New connects to a server over stream and starts listening for what it sends.
func New(t *testing.T, ctx context.Context, stream io.ReadWriteCloser) *Client {
	t.Helper()

	client := &Client{
		ctx:       ctx,
		conn:      jsonrpc2.NewConn(jsonrpc2.NewStream(stream), jsonrpc2.WithCodec(lspcodec.Codec{})),
		timeout:   DefaultTimeout,
		published: map[uri.URI]publication{},
		consumed:  map[uri.URI]int{},
	}

	client.conn.Go(ctx, client.handle)

	t.Cleanup(func() {
		_ = client.conn.Close()
	})

	return client
}

// --- Requests ---

// Initialize performs the handshake, rooted at the workspace directory.
func (c *Client) Initialize(rootURI uri.URI) (*protocol.InitializeResult, error) {
	options, err := protocol.Marshal(map[string]any{
		"isCciExtension": true,
	})
	if err != nil {
		return nil, err
	}

	params := protocol.InitializeParams{
		Capabilities:          protocol.ClientCapabilities{},
		InitializationOptions: options,
	}
	params.WorkspaceFolders = protocol.NewNullable([]protocol.WorkspaceFolder{
		{URI: rootURI, Name: filepath.Base(rootURI.FsPath())},
	})

	result := &protocol.InitializeResult{}
	if err := c.call(protocol.MethodInitialize, params, result); err != nil {
		return nil, err
	}

	if err := c.conn.Notify(c.ctx, protocol.MethodInitialized, protocol.InitializedParams{}); err != nil {
		return nil, err
	}

	return result, nil
}

// DidOpen tells the server a document is open, which is what starts the work
// that ends in a diagnostics notification.
func (c *Client) DidOpen(docURI uri.URI, content string) error {
	return c.conn.Notify(c.ctx, protocol.MethodTextDocumentDidOpen, protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        docURI,
			LanguageID: "yaml",
			Version:    1,
			Text:       content,
		},
	})
}

// DidChange replaces a document's content. The version has to advance, because
// the server drops diagnostics computed for a version it no longer holds.
func (c *Client) DidChange(docURI uri.URI, version int32, content string) error {
	return c.conn.Notify(c.ctx, protocol.MethodTextDocumentDidChange, protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: docURI},
			Version:                version,
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			&protocol.TextDocumentContentChangeWholeDocument{Text: content},
		},
	})
}

// DidClose tells the server a document is closed.
func (c *Client) DidClose(docURI uri.URI) error {
	return c.conn.Notify(c.ctx, protocol.MethodTextDocumentDidClose, protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
	})
}

// Completion asks what could be written at a position.
func (c *Client) Completion(docURI uri.URI, position protocol.Position) (*protocol.CompletionList, error) {
	params := protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
			Position:     position,
		},
	}

	result := &protocol.CompletionList{}
	if err := c.call(protocol.MethodTextDocumentCompletion, params, result); err != nil {
		return nil, err
	}

	return result, nil
}

// Hover asks what is at a position.
func (c *Client) Hover(docURI uri.URI, position protocol.Position) (*protocol.Hover, error) {
	params := protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
			Position:     position,
		},
	}

	result := &protocol.Hover{}
	if err := c.call(protocol.MethodTextDocumentHover, params, result); err != nil {
		return nil, err
	}

	return result, nil
}

// ExecuteCommand runs one of the server's commands, which is how a client
// configures the host and token it should use.
func (c *Client) ExecuteCommand(command string, arguments ...any) error {
	params := protocol.ExecuteCommandParams{
		Command: command,
	}
	for _, argument := range arguments {
		encoded, err := protocol.Marshal(argument)
		if err != nil {
			return err
		}
		params.Arguments = append(params.Arguments, encoded)
	}

	return c.call(protocol.MethodWorkspaceExecuteCommand, params, nil)
}

// Call makes a request the client has no method of its own for.
func (c *Client) Call(method string, params, result any) error {
	return c.call(method, params, result)
}

// --- Notifications the server sends ---

// WaitForDiagnostics waits for the next diagnostics the server publishes for a
// document and returns them, marking them read: a later call waits for the
// publication after this one, so a test can follow a revalidation.
func (c *Client) WaitForDiagnostics(docURI uri.URI) ([]protocol.Diagnostic, error) {
	return c.WaitForDiagnosticsWithin(docURI, c.timeout)
}

// WaitForDiagnosticsWithin is WaitForDiagnostics with an explicit bound, for a
// case that expects to wait longer, or to give up sooner.
func (c *Client) WaitForDiagnosticsWithin(docURI uri.URI, timeout time.Duration) ([]protocol.Diagnostic, error) {
	deadline := time.Now().Add(timeout)

	for {
		c.mu.Lock()
		latest, published := c.published[docURI]
		unread := published && latest.count > c.consumed[docURI]
		if unread {
			c.consumed[docURI] = latest.count
		}
		c.mu.Unlock()

		if unread {
			return latest.items, nil
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("no diagnostics published for %s within %s", docURI, timeout)
		}

		time.Sleep(pollInterval)
	}
}

// Telemetry is every telemetry event the server has sent. The server reports
// its own activity this way, so it doubles as a record of what it did.
func (c *Client) Telemetry() []map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()

	return append([]map[string]any(nil), c.telemetry...)
}

// LogMessages is every window/logMessage the server has sent.
func (c *Client) LogMessages() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return append([]string(nil), c.logMessages...)
}

// --- Plumbing ---

func (c *Client) call(method string, params, result any) error {
	ctx, cancel := context.WithTimeout(c.ctx, c.timeout)
	defer cancel()

	if _, err := c.conn.Call(ctx, method, params, result); err != nil {
		return fmt.Errorf("%s: %w", method, err)
	}

	return nil
}

// handle takes what the server sends. Everything it sends is a notification,
// so nothing here needs a reply; a request the client does not implement is
// answered as unhandled rather than ignored, so that adding one server-side
// does not silently stall. A notification that does not decode closes the
// connection, so the test's next request fails rather than going on without
// it.
func (c *Client) handle(ctx context.Context, req *jsonrpc2.Request) (any, error) {
	switch req.Method() {
	case protocol.MethodTextDocumentPublishDiagnostics:
		var params protocol.PublishDiagnosticsParams
		if err := protocol.Unmarshal(req.Params(), &params); err != nil {
			return nil, err
		}
		c.recordDiagnostics(params)

		return nil, nil

	case protocol.MethodTelemetryEvent:
		var event map[string]any
		if err := protocol.Unmarshal(req.Params(), &event); err != nil {
			return nil, err
		}
		c.mu.Lock()
		c.telemetry = append(c.telemetry, event)
		c.mu.Unlock()

		return nil, nil

	case protocol.MethodWindowLogMessage, protocol.MethodWindowShowMessage:
		var params protocol.LogMessageParams
		if err := protocol.Unmarshal(req.Params(), &params); err != nil {
			return nil, err
		}
		c.mu.Lock()
		c.logMessages = append(c.logMessages, params.Message)
		c.mu.Unlock()

		return nil, nil

	default:
		return jsonrpc2.MethodNotFoundHandler(ctx, req)
	}
}

func (c *Client) recordDiagnostics(params protocol.PublishDiagnosticsParams) {
	c.mu.Lock()
	defer c.mu.Unlock()

	previous := c.published[params.URI]
	c.published[params.URI] = publication{
		count: previous.count + 1,
		items: params.Diagnostics,
	}
}
