package methods

import (
	"context"

	"go.lsp.dev/protocol"

	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
)

func (methods *Methods) PrepareRename(_ context.Context, params *protocol.PrepareRenameParams) (protocol.PrepareRenameResult, error) {
	placeholder, err := languageservice.PrepareRename(*params, methods.Cache, methods.Settings())
	if placeholder == nil || err != nil {
		return nil, err
	}
	return placeholder, nil
}

func (methods *Methods) Rename(_ context.Context, params *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
	return languageservice.Rename(*params, methods.Cache, methods.Settings())
}
