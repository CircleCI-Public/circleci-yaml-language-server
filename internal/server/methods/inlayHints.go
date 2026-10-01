package methods

import (
	"context"
	"log/slog"

	"go.lsp.dev/protocol"

	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
)

func (methods *Methods) InlayHint(_ context.Context, params *protocol.InlayHintParams) ([]protocol.InlayHint, error) {
	return languageservice.InlayHints(*params, methods.Cache, methods.Settings())
}

// refreshInlayHints asks the client to ask for inlay hints again, for a client
// that can be asked. Hints show what is known of an orb once it is fetched,
// and an edit can fetch one after the client has asked for the edit's hints.
func (methods *Methods) refreshInlayHints() {
	if !methods.inlayHintRefresh {
		return
	}
	if err := methods.Client.InlayHintRefresh(methods.Ctx); err != nil {
		slog.Debug("refreshing inlay hints", "err", err)
	}
}

func inlayHintRefresh(capabilities protocol.ClientCapabilities) bool {
	workspace := capabilities.Workspace
	if workspace == nil || workspace.InlayHint == nil || workspace.InlayHint.RefreshSupport == nil {
		return false
	}
	return *workspace.InlayHint.RefreshSupport
}
