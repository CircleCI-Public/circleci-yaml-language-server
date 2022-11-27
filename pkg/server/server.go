package languageserver

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	methods "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/server/methods"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

type JSONRPCServer struct {
	ctx            context.Context
	conn           jsonrpc2.Conn
	methods        methods.Methods
	cache          *utils.Cache
	SchemaLocation string
}

func (server JSONRPCServer) commandHandler(_ context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	fmt.Println("Called method: " + req.Method())

	switch req.Method() {

	case protocol.MethodInitialize:
		return server.methods.Initialize(reply)

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

	case protocol.MethodShutdown:
		return reply(server.ctx, nil, nil)

	case protocol.MethodExit:
		os.Exit(0)
		return nil

	default:
		return jsonrpc2.MethodNotFoundHandler(server.ctx, reply, req)
	}
}

func (server JSONRPCServer) ServeStream(_ context.Context, conn jsonrpc2.Conn) error {
	server.conn = conn
	server.cache = utils.CreateCache()
	server.methods = methods.Methods{
		Ctx:            server.ctx,
		Conn:           server.conn,
		Cache:          server.cache,
		SchemaLocation: server.SchemaLocation,
	}
	conn.Go(server.ctx, server.commandHandler)
	<-conn.Done()
	return conn.Err()
}

func StartServer(port int, host string, schema string) {
	ctx := context.Background()
	// The LSP client waits that the server prints "Server started" on stdout to connect. The best
	// solution would be to make this the "express way" and give a callback to ListenAndServe that
	// would print the "Server started" but it seems that doesn't exist in go
	// https://stackoverflow.com/questions/34312615/log-when-server-is-started
	// So we just print the log one second after the server started
	go func() {
		time.Sleep(1 * time.Second)
		fmt.Printf("Server started on port %d, version %s\n", port, methods.ServerVersion)
		fmt.Printf("   JSON Schema: %s", schema)
	}()

	err := jsonrpc2.ListenAndServe(
		ctx,
		"tcp",
		fmt.Sprintf("%s:%d", host, port),
		JSONRPCServer{
			ctx:            ctx,
			SchemaLocation: schema,
		},
		0,
	)

	if err != nil {
		panic(err)
	}
}

type StdioReadWriteCloser struct {
	io.Reader
	io.Writer
}

func (s *StdioReadWriteCloser) Close() error { return nil }

func StartServerStdio(schema string) {
	ctx := context.Background()

	stdioStream := jsonrpc2.NewStream(&StdioReadWriteCloser{os.Stdin, os.Stdout})
	stdioConn := jsonrpc2.NewConn(stdioStream)
	server := JSONRPCServer{
		ctx:            ctx,
		SchemaLocation: schema,
	}

	if err := server.ServeStream(ctx, stdioConn); err != nil {
		panic(err)
	}
}

func GetServerVersion() string {
	return methods.ServerVersion
}
