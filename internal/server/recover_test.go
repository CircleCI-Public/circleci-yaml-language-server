package languageserver

import (
	"context"
	"errors"
	"testing"

	"go.lsp.dev/jsonrpc2"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

// serve runs handler on one end of an in-memory connection, and returns the
// other end to call it from.
func serve(t *testing.T, handler jsonrpc2.Handler) jsonrpc2.Conn {
	t.Helper()

	serverStream, clientStream := jsonrpc2.NewChannelStreamPair(0)
	server := jsonrpc2.NewConn(serverStream)
	server.Go(context.Background(), handler)
	client := jsonrpc2.NewConn(clientStream)
	client.Go(context.Background(), jsonrpc2.MethodNotFoundHandler)
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})

	return client
}

func TestRecoverPanics(t *testing.T) {
	ctx := context.Background()

	handler := func(_ context.Context, req *jsonrpc2.Request) (any, error) {
		switch req.Method() {
		case "panic", "panicNotification":
			panic(errors.New("boom"))
		default:
			return "ok", nil
		}
	}

	t.Run("answers a panicking request with an internal error", func(t *testing.T) {
		client := serve(t, recoverPanics(handler))

		_, err := client.Call(ctx, "panic", nil, nil)

		var rpcErr *jsonrpc2.Error
		assert.Assert(t, errors.As(err, &rpcErr), "error %v is not a JSON-RPC error", err)
		assert.Check(t, cmp.Equal(rpcErr.Code, jsonrpc2.InternalError))
		assert.Check(t, cmp.Equal(rpcErr.Message, "panic: internal error"))

		t.Run("and keeps the connection open", func(t *testing.T) {
			var result string
			_, err := client.Call(ctx, "fine", nil, &result)
			assert.Check(t, err)
			assert.Check(t, cmp.Equal(result, "ok"))
		})
	})

	t.Run("keeps the connection open after a panicking notification", func(t *testing.T) {
		client := serve(t, recoverPanics(handler))

		assert.Check(t, client.Notify(ctx, "panicNotification", nil))

		var result string
		_, err := client.Call(ctx, "fine", nil, &result)
		assert.Check(t, err)
		assert.Check(t, cmp.Equal(result, "ok"))
	})

	t.Run("passes a request that does not panic through", func(t *testing.T) {
		client := serve(t, recoverPanics(handler))

		var result string
		_, err := client.Call(ctx, "fine", nil, &result)
		assert.Check(t, err)
		assert.Check(t, cmp.Equal(result, "ok"))
	})
}

func TestDropFailedNotifications(t *testing.T) {
	ctx := context.Background()

	handler := dropFailedNotifications(func(_ context.Context, req *jsonrpc2.Request) (any, error) {
		switch req.Method() {
		case "fail", "failNotification":
			return nil, errors.New("boom")
		default:
			return "ok", nil
		}
	})

	t.Run("keeps the connection open after a failing notification", func(t *testing.T) {
		client := serve(t, handler)

		assert.Check(t, client.Notify(ctx, "failNotification", nil))

		var result string
		_, err := client.Call(ctx, "fine", nil, &result)
		assert.Check(t, err)
		assert.Check(t, cmp.Equal(result, "ok"))
	})

	t.Run("still answers a failing request with its error", func(t *testing.T) {
		client := serve(t, handler)

		_, err := client.Call(ctx, "fail", nil, nil)
		assert.Check(t, cmp.ErrorContains(err, "boom"))
	})
}
