package languageservice

import (
	"errors"

	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/services/documentSymbols"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func DocumentSymbols(params protocol.DocumentSymbolParams, cache *utils.Cache, context *utils.LsContext) ([]protocol.DocumentSymbol, error) {
	yamlDocument, err := yamlparser.ParseFromUriWithCache(params.TextDocument.URI, cache, context)

	if errors.Is(err, yamlparser.CacheMissingError) {
		yamlDocument, err = yamlparser.ParseFromURI(params.TextDocument.URI, context)
	}

	if err != nil {
		return nil, err
	}

	symbols := documentSymbols.SymbolsForDocument(&yamlDocument)

	return symbols, nil
}
