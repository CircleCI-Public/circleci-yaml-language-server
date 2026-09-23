package languageservice

import (
	"errors"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/services/documentSymbols"
)

func DocumentSymbols(params protocol.DocumentSymbolParams, cache *cache.Cache, context *session.Settings) ([]protocol.DocumentSymbol, error) {
	yamlDocument, err := yamlparser.ParseFromUriWithCache(params.TextDocument.URI, cache, context)

	if errors.Is(err, yamlparser.ErrCacheMissing) {
		yamlDocument, err = yamlparser.ParseFromURI(params.TextDocument.URI, context)
	}

	if err != nil {
		return nil, err
	}
	defer yamlDocument.Close()

	symbols := documentSymbols.SymbolsForDocument(&yamlDocument)

	return symbols, nil
}
