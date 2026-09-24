package methods

import (
	"context"
	"fmt"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

type Methods struct {
	Ctx            context.Context
	Conn           jsonrpc2.Conn
	Cache          *cache.Cache
	Settings       *session.Settings
	SchemaLocation string
}

// decode reads a request's params into the protocol type T. The codec is the
// protocol's own, which is what reads its union and optional fields.
func decode[T any](raw jsonrpc2.RawMessage) (T, error) {
	var params T
	if err := protocol.Unmarshal(raw, &params); err != nil {
		return params, fmt.Errorf("%s: %w", jsonrpc2.ErrParse, err)
	}
	return params, nil
}
