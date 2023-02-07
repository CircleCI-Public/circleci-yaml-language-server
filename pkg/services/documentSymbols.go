package languageservice

import (
	"fmt"

	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/services/documentSymbols"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func DocumentSymbols(params protocol.DocumentSymbolParams, cache *utils.Cache, context *utils.LsContext) ([]protocol.DocumentSymbol, error) {
	fmt.Println("LanguageServer.DocumentSymbols")

	yamlDocument, err := yamlparser.ParseFromUriWithCache(params.TextDocument.URI, cache, context)

	if err != nil {
		return nil, err
	}

	symbols := documentSymbols.SymbolsForDocument(&yamlDocument)

	return symbols, nil
}
