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
// error, rather than letting the panic take the session down: one broken
// request should fail on its own, not end the session for every open
// document.
//
// The connection recovers a handler's panic itself, but then closes the
// connection, so the panic has to be caught before it gets there. A
// notification has no reply to carry an error, and returning one also closes
// the connection, so its panic is only logged.
func recoverPanics(handler jsonrpc2.Handler) jsonrpc2.Handler {
	return func(ctx context.Context, req *jsonrpc2.Request) (result any, err error) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			slog.Error("request panicked", "method", req.Method(), "panic", recovered, "stack", string(debug.Stack()))
			rollbar.LogPanic(recovered, false)
			result = nil
			err = nil
			if req.IsCall() {
				err = jsonrpc2.NewError(jsonrpc2.InternalError, fmt.Sprintf("%s: internal error", req.Method()))
			}
		}()
		return handler(ctx, req)
	}
}
