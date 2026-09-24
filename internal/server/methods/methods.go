package methods

import (
	"context"
	"os"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

// Methods is the language server: protocol.ServerHandler decodes each request
// and calls the method for it. A request the server does not handle is left to
// UnimplementedServer, which answers method-not-found, or ignores it if it is
// a notification.
type Methods struct {
	protocol.UnimplementedServer

	Ctx            context.Context
	Client         protocol.Client
	Cache          *cache.Cache
	Settings       *session.Settings
	SchemaLocation string
}

var _ protocol.Server = (*Methods)(nil)

func (methods *Methods) Shutdown(context.Context) error {
	return nil
}

func (methods *Methods) Exit(context.Context) error {
	os.Exit(0)
	return nil
}
