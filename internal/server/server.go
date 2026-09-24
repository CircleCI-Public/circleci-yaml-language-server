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
// Requests are handled one at a time, in the order they arrive.
func (server JSONRPCServer) serve(stream jsonrpc2.Stream) error {
	defer rollbar.Close()
	slog.Info("new client connection")

	conn := jsonrpc2.NewConn(stream, jsonrpc2.WithCodec(lspcodec.Codec{}))
	lsp := &methods.Methods{
		Ctx:            server.ctx,
		Client:         protocol.ClientDispatcher(conn),
		Cache:          cache.New(),
		Settings:       server.lsContext,
		SchemaLocation: server.SchemaLocation,
	}
	handler := protocol.ServerHandler(lsp, jsonrpc2.MethodNotFoundHandler)
	conn.Go(server.ctx, recoverPanics(dropFailedNotifications(logMethods(handler))))
	<-conn.Done()

	return conn.Err()
}

func logMethods(handler jsonrpc2.Handler) jsonrpc2.Handler {
	return func(ctx context.Context, req *jsonrpc2.Request) (any, error) {
		slog.Debug("called method", "method", req.Method())
		return handler(ctx, req)
	}
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

	// jsonrpc2.Serve would build each connection itself, with a codec that
	// cannot write the protocol's types, so connections are accepted here.
	for {
		netConn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		go func() {
			stream := jsonrpc2.NewStream(netConn)
			if err := server.serve(stream); err != nil {
				slog.Info("client connection closed", "err", err)
			}
			_ = stream.Close()
		}()
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
	server := getJsonRpcServer(ctx, schemaLocation)

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
