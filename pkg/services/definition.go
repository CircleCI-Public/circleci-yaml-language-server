package languageservice

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/services/definition"
	"go.lsp.dev/protocol"
)

func Definition(params protocol.DefinitionParams, cache *cache.Cache, context *session.Settings) ([]protocol.Location, error) {
	yamlDocument, err := yamlparser.ParseFromUriWithCache(params.TextDocument.URI, cache, context)
	if err != nil {
		return nil, err
	}

	def := definition.DefinitionStruct{Cache: cache, Params: params, Doc: yamlDocument}

	return def.Definition()
}
