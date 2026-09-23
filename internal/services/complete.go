package languageservice

import (
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/services/complete"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

func Complete(params protocol.CompletionParams, cache *cache.Cache, context *session.Settings) (protocol.CompletionList, error) {
	yamlDocument, err := yamlparser.ParseFromUriWithCache(params.TextDocument.URI, cache, context)

	if err != nil {
		return protocol.CompletionList{}, err
	}
	defer yamlDocument.Close()

	if yamlDocument.Version < 2.1 {
		return protocol.CompletionList{
			IsIncomplete: true,
			Items:        []protocol.CompletionItem{},
		}, nil
	}

	completionHandler := complete.CompletionHandler{
		Params:  params,
		Doc:     yamlDocument,
		Cache:   cache,
		Items:   []protocol.CompletionItem{},
		Context: context,
	}
	completionHandler.GetCompletionItems()

	return protocol.CompletionList{
		IsIncomplete: true,
		Items:        completionHandler.Items,
	}, nil
}
