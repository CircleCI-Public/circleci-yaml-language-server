package languageservice

import (
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/services/definition"
	utils "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func Definition(params protocol.DefinitionParams, cache *utils.Cache) ([]protocol.Location, error) {
	yamlDocument, err := yamlparser.GetParsedYAMLWithCache(params.TextDocument.URI, cache)
	if err != nil {
		return nil, err
	}

	def := definition.DefinitionStruct{Cache: cache, Params: params, Doc: yamlDocument}

	return def.Definition(yamlDocument)
}
