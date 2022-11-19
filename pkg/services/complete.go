package languageservice

import (
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/services/complete"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"

	"go.lsp.dev/protocol"
)

func Complete(params protocol.CompletionParams, cache *utils.Cache) (protocol.CompletionList, error) {
	yamlDocument, err := yamlparser.ParseFileWithCache(params.TextDocument.URI, cache)

	if err != nil {
		return protocol.CompletionList{}, err
	}

	if yamlDocument.Version < 2.1 {
		return protocol.CompletionList{
			IsIncomplete: true,
			Items:        []protocol.CompletionItem{},
		}, nil
	}

	completionHandler := complete.CompletionHandler{
		Params: params,
		Doc:    yamlDocument,
		Cache:  cache,
		Items:  []protocol.CompletionItem{},
	}
	completionHandler.GetCompletionItems()

	return protocol.CompletionList{
		IsIncomplete: true,
		Items:        completionHandler.Items,
	}, nil
}
