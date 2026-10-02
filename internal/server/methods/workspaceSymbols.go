package methods

import (
	"context"
	"io/fs"
	"path/filepath"
	"slices"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
)

func (methods *Methods) Symbols(ctx context.Context, params *protocol.WorkspaceSymbolParams) (protocol.WorkspaceSymbolResult, error) {
	symbols := languageservice.WorkspaceSymbols(ctx, params.Query, methods.workspaceConfigs(), methods.Cache, methods.Settings())
	return protocol.SymbolInformationSlice(symbols), nil
}

// workspaceConfigs are the config files open in the editor, and the YAML in
// each workspace folder's .circleci directory, whether open or not. An orb's
// source the server wrote out isn't one.
func (methods *Methods) workspaceConfigs() []uri.URI {
	configs := []uri.URI{}
	for document := range methods.Cache.FileCache.GetFiles() {
		if inCircleCIDirectory(document) {
			configs = append(configs, document)
		}
	}

	for _, folder := range methods.workspaceFolders {
		root := filepath.Join(folder.FsPath(), ".circleci")
		_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil
			}
			if ext := filepath.Ext(path); ext != ".yml" && ext != ".yaml" {
				return nil
			}
			if document := uri.File(path); !slices.Contains(configs, document) {
				configs = append(configs, document)
			}
			return nil
		})
	}

	return configs
}
