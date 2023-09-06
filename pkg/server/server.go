package languageserver

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	methods "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/server/methods"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"github.com/rollbar/rollbar-go"
)

type JSONRPCServer struct {
	ctx            context.Context
	conn           jsonrpc2.Conn
	methods        methods.Methods
	cache          *utils.Cache
	lsContext      *utils.LsContext
	SchemaLocation string
}

func (server JSONRPCServer) commandHandler(_ context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	fmt.Println("Called method: " + req.Method())

	defer func() {
		err := recover()
		rollbar.LogPanic(err, true)

		if err != nil {
			panic(err)
		}
	}()

	switch req.Method() {

	case protocol.MethodInitialize:
		return server.methods.Initialize(reply, req)

	case protocol.MethodWorkspaceExecuteCommand:
		return server.methods.ExecuteCommand(reply, req)

	case protocol.MethodTextDocumentDidOpen:
		return server.methods.DidOpen(reply, req)

	case protocol.MethodTextDocumentDidClose:
		return server.methods.DidClose(reply, req)

	case protocol.MethodTextDocumentDidChange:
		return server.methods.DidChange(reply, req)

	case protocol.MethodTextDocumentHover:
		return server.methods.Hover(reply, req)

	case protocol.MethodSemanticTokensFull:
		return server.methods.SemanticTokens(reply, req)

	case protocol.MethodTextDocumentDefinition:
		return server.methods.Definition(reply, req)

	case protocol.MethodTextDocumentReferences:
		return server.methods.References(reply, req)

	case protocol.MethodTextDocumentCompletion:
		return server.methods.Complete(reply, req)

	case protocol.MethodTextDocumentCodeAction:
		return server.methods.CodeAction(reply, req)

	case protocol.MethodShutdown:
		return reply(server.ctx, nil, nil)

	case protocol.MethodTextDocumentDocumentSymbol:
		return server.methods.DocumentSymbols(reply, req)

	case protocol.MethodExit:
		os.Exit(0)
		return nil

	default:
		return jsonrpc2.MethodNotFoundHandler(server.ctx, reply, req)
	}
}

func (server JSONRPCServer) ServeStream(_ context.Context, conn jsonrpc2.Conn) error {
	fmt.Println("New client connection")

	server.conn = conn
	server.cache = utils.CreateCache()
	server.methods = methods.Methods{
		Ctx:            server.ctx,
		Conn:           server.conn,
		Cache:          server.cache,
		LsContext:      server.lsContext,
		SchemaLocation: server.SchemaLocation,
	}
	conn.Go(server.ctx, server.commandHandler)
	<-conn.Done()

	rollbar.Close()

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

	// The LSP client waits that the server prints "Server started" on stdout to connect. The best
	// solution would be to make this the "express way" and give a callback to ListenAndServe that
	// would print the "Server started" but it seems that doesn't exist in go
	// https://stackoverflow.com/questions/34312615/log-when-server-is-started
	// So we just print the log one second after the server started
	go func() {
		time.Sleep(1 * time.Second)
		fmt.Printf("Server started on port %d, version %s\n", port, methods.ServerVersion)
		fmt.Printf("   JSON Schema: %s", schemaLocation)
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
		lsContext: &utils.LsContext{
			Api: utils.ApiContext{
				HostUrl: utils.CIRCLE_CI_APP_HOST_URL,
				Token:   "",
			},
			IsCciExtension: false,
		},
		SchemaLocation: schemaLocation,
	}
}

func GetServerVersion() string {
	return methods.ServerVersion
}
