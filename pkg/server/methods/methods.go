package methods

import (
	"context"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/jsonrpc2"
)

type Methods struct {
	Ctx            context.Context
	Conn           jsonrpc2.Conn
	Cache          *utils.Cache
	SchemaLocation string
}
