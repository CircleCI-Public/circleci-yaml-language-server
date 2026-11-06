package methods

import (
	"context"
	"net"
	"testing"

	"go.lsp.dev/jsonrpc2"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/lspcodec"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

// withClientAnswering returns methods whose client answers every request with
// result, or with err when it isn't nil.
func withClientAnswering(t *testing.T, result any, err error) *Methods {
	t.Helper()

	serverEnd, clientEnd := net.Pipe()
	serverConn := jsonrpc2.NewConn(jsonrpc2.NewStream(serverEnd), jsonrpc2.WithCodec(lspcodec.Codec{}))
	clientConn := jsonrpc2.NewConn(jsonrpc2.NewStream(clientEnd), jsonrpc2.WithCodec(lspcodec.Codec{}))
	serverConn.Go(t.Context(), jsonrpc2.MethodNotFoundHandler)
	clientConn.Go(t.Context(), func(context.Context, *jsonrpc2.Request) (any, error) {
		return result, err
	})
	t.Cleanup(func() {
		_ = clientConn.Close()
		_ = serverConn.Close()
	})

	return New(t.Context(), serverConn, cache.New(), session.Settings{}, "")
}

func TestFetchGitHubFile(t *testing.T) {
	const address = "https://raw.githubusercontent.com/acme/orbs/main/go.yml"

	t.Run("a source is found", func(t *testing.T) {
		methods := withClientAnswering(t, map[string]any{"found": true, "source": "version: 2.1\n"}, nil)
		source, found, err := methods.fetchGitHubFile(t.Context(), address)
		assert.Check(t, err)
		assert.Check(t, found)
		assert.Check(t, cmp.Equal(source, "version: 2.1\n"))
	})

	t.Run("an empty source is found", func(t *testing.T) {
		methods := withClientAnswering(t, map[string]any{"found": true, "source": ""}, nil)
		_, found, err := methods.fetchGitHubFile(t.Context(), address)
		assert.Check(t, err)
		assert.Check(t, found)
	})

	t.Run("not found isn't there", func(t *testing.T) {
		methods := withClientAnswering(t, map[string]any{"found": false}, nil)
		_, found, err := methods.fetchGitHubFile(t.Context(), address)
		assert.Check(t, err)
		assert.Check(t, !found)
	})

	for name, result := range map[string]any{
		"null":                nil,
		"found but no source": map[string]any{"found": true},
	} {
		t.Run("an answer of "+name+" is an error, not missing", func(t *testing.T) {
			methods := withClientAnswering(t, result, nil)
			_, found, err := methods.fetchGitHubFile(t.Context(), address)
			assert.Check(t, cmp.ErrorIs(err, errMalformedGitHubFile))
			assert.Check(t, !found)
		})
	}

	t.Run("an answer that isn't an object is an error", func(t *testing.T) {
		methods := withClientAnswering(t, "version: 2.1", nil)
		_, found, err := methods.fetchGitHubFile(t.Context(), address)
		assert.Check(t, err != nil)
		assert.Check(t, !found)
	})

	t.Run("the client failing is an error", func(t *testing.T) {
		methods := withClientAnswering(t, nil, jsonrpc2.NewError(jsonrpc2.InternalError, "GitHub unreachable"))
		_, found, err := methods.fetchGitHubFile(t.Context(), address)
		assert.Check(t, cmp.ErrorContains(err, "GitHub unreachable"))
		assert.Check(t, !found)
	})
}
