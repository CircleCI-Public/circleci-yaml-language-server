package methods

import (
	"context"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
	"go.lsp.dev/jsonrpc2"
)

type Methods struct {
	Ctx            context.Context
	Conn           jsonrpc2.Conn
	Cache          *cache.Cache
	Settings       *session.Settings
	SchemaLocation string
}
