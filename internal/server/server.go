package languageserver

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"time"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	"github.com/rollbar/rollbar-go"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/server/methods"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/version"
)

type JSONRPCServer struct {
	ctx            context.Context
	conn           jsonrpc2.Conn
	methods        methods.Methods
	cache          *cache.Cache
	lsContext      *session.Settings
	SchemaLocation string
}

func (server JSONRPCServer) commandHandler(_ context.Context, req *jsonrpc2.Request) (any, error) {
	slog.Debug("called method", "method", req.Method())

	params := req.Params()

	switch req.Method() {

	case protocol.MethodInitialize:
		return server.methods.Initialize(params)

	case protocol.MethodWorkspaceExecuteCommand:
		return server.methods.ExecuteCommand(params)

	case protocol.MethodTextDocumentDidOpen:
		server.methods.DidOpen(params)
		return nil, nil

	case protocol.MethodTextDocumentDidClose:
		server.methods.DidClose(params)
		return nil, nil

	case protocol.MethodTextDocumentDidChange:
		server.methods.DidChange(params)
		return nil, nil

	case protocol.MethodTextDocumentHover:
		return server.methods.Hover(params)

	case protocol.MethodTextDocumentSemanticTokensFull:
		return server.methods.SemanticTokens(params)

	case protocol.MethodTextDocumentDefinition:
		return server.methods.Definition(params)

	case protocol.MethodTextDocumentReferences:
		return server.methods.References(params)

	case protocol.MethodTextDocumentCompletion:
		return server.methods.Complete(params)

	case protocol.MethodTextDocumentCodeAction:
		return server.methods.CodeAction(params)

	case protocol.MethodShutdown:
		return nil, nil

	case protocol.MethodTextDocumentDocumentSymbol:
		return server.methods.DocumentSymbols(params)

	case protocol.MethodExit:
		os.Exit(0)
		return nil, nil

	default:
		// A notification we do not handle is dropped: an error from one
		// closes the connection.
		if !req.IsCall() {
			return nil, nil
		}
		return nil, jsonrpc2.ErrMethodNotFound
	}
}

// encodeResults encodes what a handler answers with using the protocol's own
// codec, which is what writes its union and optional fields. It is done here
// rather than by the connection's codec because a connection jsonrpc2.Serve
// accepts always has the default one.
func encodeResults(handler jsonrpc2.Handler) jsonrpc2.Handler {
	return func(ctx context.Context, req *jsonrpc2.Request) (any, error) {
		result, err := handler(ctx, req)
		if err != nil || result == nil {
			return nil, err
		}
		encoded, err := protocol.Marshal(result)
		if err != nil {
			return nil, jsonrpc2.Errorf(jsonrpc2.InternalError, "%s: encoding result: %v", req.Method(), err)
		}
		return jsonrpc2.RawMessage(encoded), nil
	}
}

func (server JSONRPCServer) ServeStream(_ context.Context, conn jsonrpc2.Conn) error {
	defer rollbar.Close()
	slog.Info("new client connection")

	server.conn = conn
	server.cache = cache.New()
	server.methods = methods.Methods{
		Ctx:            server.ctx,
		Conn:           server.conn,
		Cache:          server.cache,
		Settings:       server.lsContext,
		SchemaLocation: server.SchemaLocation,
	}
	conn.Go(server.ctx, recoverPanics(encodeResults(server.commandHandler)))
	<-conn.Done()

	return conn.Err()
}

func StartServer(port int, host string, schemaLocation string) {
	ctx := context.Background()
	server := getJsonRpcServer(ctx, schemaLocation)

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

	err = jsonrpc2.Serve(ctx, ln, server, 0)

	if err != nil {
		panic(err)
	}
}

type StdioReadWriteCloser struct {
	io.Reader
	io.Writer
}

func (s *StdioReadWriteCloser) Close() error { return nil }

func StartServerStdio(schemaLocation string) {
	ctx := context.Background()

	stdioStream := jsonrpc2.NewStream(&StdioReadWriteCloser{os.Stdin, os.Stdout})
	stdioConn := jsonrpc2.NewConn(stdioStream)
	server := getJsonRpcServer(ctx, schemaLocation)

	if err := server.ServeStream(ctx, stdioConn); err != nil {
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
				// A self-hosted install can serve the runner API somewhere
				// other than runner.<host>. The host and token are set later
				// over the protocol; this one has no command, so it is read
				// from the environment.
				RunnerHost: os.Getenv("CIRCLECI_RUNNER_HOST"),
			},
			IsCciExtension: false,
		},
		SchemaLocation: schemaLocation,
	}
}
