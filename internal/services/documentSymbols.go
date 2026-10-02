package languageservice

import (
	"context"
	"errors"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/services/documentSymbols"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

func DocumentSymbols(ctx context.Context, params protocol.DocumentSymbolParams, cache *cache.Cache, context *session.Settings) ([]protocol.DocumentSymbol, error) {
	yamlDocument, err := yamlparser.ParseFromUriWithCache(ctx, params.TextDocument.URI, cache, context)

	if errors.Is(err, yamlparser.ErrCacheMissing) {
		yamlDocument, err = yamlparser.ParseFromURI(ctx, params.TextDocument.URI, context)
	}

	if err != nil {
		return nil, err
	}
	defer yamlDocument.Close()

	symbols := documentSymbols.SymbolsForDocument(&yamlDocument)

	return symbols, nil
}
