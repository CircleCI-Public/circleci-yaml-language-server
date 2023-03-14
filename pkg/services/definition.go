package languageservice

import (
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/services/definition"
	utils "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func Definition(params protocol.DefinitionParams, cache *utils.Cache, context *utils.LsContext) ([]protocol.Location, error) {
	yamlDocument, err := yamlparser.ParseFromUriWithCache(params.TextDocument.URI, cache, context)
	if err != nil {
		return nil, err
	}

	def := definition.DefinitionStruct{Cache: cache, Params: params, Doc: yamlDocument}

	return def.Definition()
}
