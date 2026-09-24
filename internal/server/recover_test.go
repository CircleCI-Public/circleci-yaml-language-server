package languageserver

import (
	"context"
	"errors"
	"testing"

	"go.lsp.dev/jsonrpc2"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

// reply is what a handler answered, as the replier saw it.
type reply struct {
	count  int
	result any
	err    error
}

func (r *reply) replier(_ context.Context, result any, err error) error {
	r.count++
	r.result = result
	r.err = err
	return nil
}

func call(t *testing.T) jsonrpc2.Request {
	t.Helper()

	req, err := jsonrpc2.NewCall(jsonrpc2.NewNumberID(1), "textDocument/completion", nil)
	assert.NilError(t, err)

	return req
}

func TestRecoverPanics(t *testing.T) {
	t.Run("answers a panicking request with an internal error", func(t *testing.T) {
		got := &reply{}
		handler := recoverPanics(func(context.Context, jsonrpc2.Replier, jsonrpc2.Request) error {
			panic(errors.New("boom"))
		})

		err := handler(context.Background(), got.replier, call(t))

		assert.Check(t, err, "a returned error would close the connection")
		assert.Check(t, cmp.Equal(got.count, 1))
		assert.Check(t, cmp.Nil(got.result))

		var rpcErr *jsonrpc2.Error
		assert.Assert(t, errors.As(got.err, &rpcErr), "reply error %v is not a JSON-RPC error", got.err)
		assert.Check(t, cmp.Equal(rpcErr.Code, jsonrpc2.InternalError))
		assert.Check(t, cmp.Equal(rpcErr.Message, "textDocument/completion: internal error"))
	})

	t.Run("does not reply twice to a request answered before it panicked", func(t *testing.T) {
		got := &reply{}
		handler := recoverPanics(func(ctx context.Context, reply jsonrpc2.Replier, _ jsonrpc2.Request) error {
			_ = reply(ctx, "answered", nil)
			panic("after replying")
		})

		err := handler(context.Background(), got.replier, call(t))

		assert.Check(t, err)
		assert.Check(t, cmp.Equal(got.count, 1))
		assert.Check(t, cmp.Equal(got.result, "answered"))
	})

	t.Run("passes a request that does not panic through", func(t *testing.T) {
		got := &reply{}
		handler := recoverPanics(func(ctx context.Context, reply jsonrpc2.Replier, _ jsonrpc2.Request) error {
			return reply(ctx, "ok", nil)
		})

		err := handler(context.Background(), got.replier, call(t))

		assert.Check(t, err)
		assert.Check(t, cmp.Equal(got.count, 1))
		assert.Check(t, cmp.Equal(got.result, "ok"))
		assert.Check(t, cmp.Nil(got.err))
	})
}
