package languageservice

import (
	"errors"
	"fmt"
	"os"

	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/services/documentSymbols"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func uriPath(uri protocol.URI) (path string, err error) {
	defer func() {
		panicErr := recover()
		if panicErr != nil {
			path = ""

			var ok bool
			if err, ok = panicErr.(error); ok {
				return
			}

			err = fmt.Errorf("%s", panicErr)
			return
		}
	}()
	path = uri.Filename()
	return
}

func DocumentSymbols(params protocol.DocumentSymbolParams, cache *utils.Cache, context *utils.LsContext) ([]protocol.DocumentSymbol, error) {
	path, err := uriPath(params.TextDocument.URI)
	if err != nil {
		return nil, err
	}

	exists := true
	_, err = os.Stat(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			// Error is not "file does not exist", stop execution and return
			return nil, err
		}
		exists = false
	}

	if !exists {
		cache.FileCache.RemoveFile(params.TextDocument.URI)
		return nil, fmt.Errorf("file does not exist: %w", err)
	}

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
