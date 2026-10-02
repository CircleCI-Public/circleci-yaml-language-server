package methods

import (
	"context"

	"go.lsp.dev/protocol"

	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
)

func (methods *Methods) PrepareRename(ctx context.Context, params *protocol.PrepareRenameParams) (protocol.PrepareRenameResult, error) {
	placeholder, err := languageservice.PrepareRename(ctx, *params, methods.Cache, methods.Settings())
	if placeholder == nil || err != nil {
		return nil, err
	}
	return placeholder, nil
}

func (methods *Methods) Rename(ctx context.Context, params *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
	return languageservice.Rename(ctx, *params, methods.Cache, methods.Settings())
}
