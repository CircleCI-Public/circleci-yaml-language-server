package languageservice

import (
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/services/definition"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

func Definition(params protocol.DefinitionParams, cache *cache.Cache, context *session.Settings) ([]protocol.Location, error) {
	yamlDocument, err := yamlparser.ParseFromUriWithCache(params.TextDocument.URI, cache, context)
	if err != nil {
		return nil, err
	}
	defer yamlDocument.Close()

	def := definition.DefinitionStruct{Cache: cache, Params: params, Doc: yamlDocument}

	return def.Definition()
}
