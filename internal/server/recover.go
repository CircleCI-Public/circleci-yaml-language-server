package languageserver

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/rollbar/rollbar-go"
	"go.lsp.dev/jsonrpc2"
)

// recoverPanics answers a request whose handler panicked with an internal
// error, rather than letting the panic take the process down: one broken
// request should fail on its own, not end the session for every open
// document.
//
// The handler runs on the connection's read loop, and an error returned from
// here closes the connection, so the panic is turned into a reply rather than
// an error.
func recoverPanics(handler jsonrpc2.Handler) jsonrpc2.Handler {
	return func(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) (err error) {
		replied := false
		tracked := func(ctx context.Context, result any, err error) error {
			replied = true
			return reply(ctx, result, err)
		}

		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			slog.Error("request panicked", "method", req.Method(), "panic", recovered, "stack", string(debug.Stack()))
			rollbar.LogPanic(recovered, false)

			if replied {
				return
			}
			err = reply(ctx, nil, jsonrpc2.NewError(jsonrpc2.InternalError, fmt.Sprintf("%s: internal error", req.Method())))
		}()

		return handler(ctx, tracked, req)
	}
}
