package languageserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/rollbar/rollbar-go"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/dockerhub"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/orburl"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/lspcodec"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/server/methods"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/version"
)

type JSONRPCServer struct {
	ctx            context.Context
	lsContext      *session.Settings
	SchemaLocation string
}

// serve runs a session with one client over stream, until the client goes.
func (server JSONRPCServer) serve(stream jsonrpc2.Stream) error {
	slog.Info("new client connection")

	conn := jsonrpc2.NewConn(stream, jsonrpc2.WithCodec(lspcodec.Codec{}))
	// Each client gets settings of its own: one setting a token should not
	// sign every other client in with it.
	lsp := methods.New(server.ctx, protocol.ClientDispatcher(conn), cache.New(), *server.lsContext, server.SchemaLocation)
	handler := protocol.ServerHandler(lsp, jsonrpc2.MethodNotFoundHandler)
	conn.Go(server.ctx, recoverPanics(dropFailedNotifications(logMethods(servedDocuments(lsp, releaseQueries(handler))))))

	select {
	case <-lsp.Exited():
	case <-conn.Done():
	case <-server.ctx.Done():
	}

	lsp.Cache.Close()

	return conn.Err()
}

// stopOnSignal returns a context that is done once the server is asked to stop
// by a signal, so that it can end its sessions and return from main rather
// than being killed where it stands.
func stopOnSignal() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

// queries are the requests that answer from what the server knows without
// changing it.
var queries = map[string]bool{
	protocol.MethodTextDocumentCodeAction:         true,
	protocol.MethodTextDocumentCompletion:         true,
	protocol.MethodTextDocumentDefinition:         true,
	protocol.MethodTextDocumentDocumentSymbol:     true,
	protocol.MethodTextDocumentHover:              true,
	protocol.MethodTextDocumentReferences:         true,
	protocol.MethodTextDocumentSemanticTokensFull: true,
	protocol.MethodWorkspaceSymbol:                true,
}

// releaseQueries lets a query run alongside the messages after it, so a slow
// one, such as a completion waiting on the network, does not hold up the rest.
// Everything else is handled in the order it arrives, and before anything
// after it starts: an edit has to apply on top of the one before, and a query
// has to see every edit and setting sent before it.
//
// That makes a query the only request still running when a message after it
// is read, so it is the only one $/cancelRequest can cancel. A query is
// registered before it is released, so a cancel read after it always finds
// it. Its context is only made cancellable once it is released, since the
// connection reuses the context of a request handled in order as soon as the
// handler returns, while a context derived from it may still be reading it.
func releaseQueries(handler jsonrpc2.Handler) jsonrpc2.Handler {
	type query struct {
		cancel    context.CancelFunc
		cancelled bool
	}
	var (
		mu      sync.Mutex
		running = map[jsonrpc2.ID]*query{}
	)

	return func(ctx context.Context, req *jsonrpc2.Request) (any, error) {
		if req.Method() == protocol.MethodCancelRequest {
			var params protocol.CancelParams
			if err := protocol.Unmarshal(req.Params(), &params); err != nil {
				return nil, err
			}
			var id jsonrpc2.ID
			switch token := params.ID.(type) {
			case protocol.Integer:
				id = jsonrpc2.NewNumberID(int64(token))
			case protocol.String:
				id = jsonrpc2.NewStringID(string(token))
			default:
				return nil, fmt.Errorf("malformed request id %v", params.ID)
			}

			mu.Lock()
			defer mu.Unlock()
			// A query that has finished, or that was never a query, has nothing
			// to cancel.
			if q, ok := running[id]; ok {
				q.cancelled = true
				if q.cancel != nil {
					q.cancel()
				}
			}
			return nil, nil
		}

		if !queries[req.Method()] || !req.IsCall() {
			return handler(ctx, req)
		}

		// The id is the request's own, which the connection reuses once the
		// query has been released.
		id := req.ID()
		if str, ok := id.StringValue(); ok {
			id = jsonrpc2.NewStringID(strings.Clone(str))
		}
		q := &query{}
		mu.Lock()
		running[id] = q
		mu.Unlock()
		defer func() {
			mu.Lock()
			delete(running, id)
			mu.Unlock()
		}()

		jsonrpc2.Async(ctx)

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		mu.Lock()
		q.cancel = cancel
		if q.cancelled {
			cancel()
		}
		mu.Unlock()

		return handler(ctx, req)
	}
}

// servedDocuments leaves out every document the session doesn't serve (see
// methods.Serves). A request about one is answered with null, which every
// textDocument request allows, and a notification about one is dropped, so
// such a document is never cached, checked or published for.
func servedDocuments(lsp *methods.Methods, handler jsonrpc2.Handler) jsonrpc2.Handler {
	return func(ctx context.Context, req *jsonrpc2.Request) (any, error) {
		if strings.HasPrefix(req.Method(), "textDocument/") {
			var params struct {
				TextDocument struct {
					URI uri.URI `json:"uri"`
				} `json:"textDocument"`
			}
			if err := json.Unmarshal(req.Params(), &params); err == nil && !lsp.Serves(params.TextDocument.URI) {
				slog.Debug("ignoring a document that isn't CircleCI config", "method", req.Method(), "uri", params.TextDocument.URI)
				return nil, nil
			}
		}
		return handler(ctx, req)
	}
}

func logMethods(handler jsonrpc2.Handler) jsonrpc2.Handler {
	return func(ctx context.Context, req *jsonrpc2.Request) (any, error) {
		slog.Debug("called method", "method", req.Method())
		return handler(ctx, req)
	}
}

func StartServer(port int, host string, schemaLocation string) {
	ctx, stop := stopOnSignal()
	defer stop()

	server := getJsonRpcServer(ctx, schemaLocation)

	// Rollbar is shared by every connection, so it is closed only once the
	// last one is over.
	defer rollbar.Close()

	if port == -1 {
		port = 0
	}

	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		panic(err)
	}

	ln, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}

	port = ln.Addr().(*net.TCPAddr).Port

	// The LSP client waits that the server prints "Server started" on stdout to connect, which is why
	// these lines are printed rather than logged: they are a handshake, not a log. The best
	// solution would be to make this the "express way" and give a callback to ListenAndServe that
	// would print the "Server started" but it seems that doesn't exist in go
	// https://stackoverflow.com/questions/34312615/log-when-server-is-started
	// So we just print the log one second after the server started
	go func() {
		time.Sleep(1 * time.Second)
		fmt.Printf("Server started on port %d, version %s\n", port, version.Server)
		if schemaLocation != "" {
			fmt.Printf("   JSON Schema: %s\n", schemaLocation)
		} else {
			fmt.Println("   JSON Schema: (embedded)")
		}
	}()

	// Stopping closes the listener, which ends the loop below.
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	// jsonrpc2.Serve would build each connection itself, with a codec that
	// cannot write the protocol's types, so connections are accepted here.
	var sessions sync.WaitGroup
	for {
		netConn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			panic(err)
		}
		sessions.Go(func() {
			stream := jsonrpc2.NewStream(netConn)
			if err := server.serve(stream); err != nil {
				slog.Info("client connection closed", "err", err)
			}
			_ = stream.Close()
		})
	}

	// Each session ends as soon as the server is stopping; wait for them to
	// clean up.
	sessions.Wait()
}

type StdioReadWriteCloser struct {
	io.Reader
	io.Writer
}

func (s *StdioReadWriteCloser) Close() error { return nil }

func StartServerStdio(schemaLocation string) {
	ctx, stop := stopOnSignal()
	defer stop()

	stdioStream := jsonrpc2.NewStream(&StdioReadWriteCloser{os.Stdin, os.Stdout})
	server := getJsonRpcServer(ctx, schemaLocation)

	defer rollbar.Close()

	if err := server.serve(stdioStream); err != nil {
		panic(err)
	}
}

func getJsonRpcServer(ctx context.Context, schemaLocation string) JSONRPCServer {
	return JSONRPCServer{
		ctx: ctx,
		lsContext: &session.Settings{
			Api: circleci.Config{
				HostUrl: circleci.DefaultHostURL,
				Token:   "",
			},
			// An editor can set this over the protocol too (setGitHubToken).
			OrbURLs: orburl.Config{GitHubToken: gitHubTokenFromEnv()},
			// Empty is the public Docker Hub. The acceptance tests set it to
			// point the server at a fake.
			DockerHub:      dockerhub.Config{BaseURL: os.Getenv("LSP_DOCKER_HUB_URL")},
			IsCciExtension: false,
		},
		SchemaLocation: schemaLocation,
	}
}

// gitHubTokenFromEnv is the GitHub token in the environment, under the names
// the GitHub CLI reads: GH_TOKEN, and then GITHUB_TOKEN.
func gitHubTokenFromEnv() string {
	if token := os.Getenv("GH_TOKEN"); token != "" {
		return token
	}
	return os.Getenv("GITHUB_TOKEN")
}
