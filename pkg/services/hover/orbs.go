package hover

import (
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

func HoverInOrbs(doc yamlparser.YamlDocument, path []string, cache *utils.Cache) string {
	if len(path) == 0 {
		return commands
	}

	orbName := path[0]
	if len(path) == 1 {
		return orbDefinition(doc, orbName, cache)
	}

	return ""
}

func orbDefinition(doc yamlparser.YamlDocument, orbName string, cache *utils.Cache) string {
	orbInDoc := doc.Orbs[orbName]
	orb := doc.GetOrbInfo(cache, orbInDoc.Name)
	if orb != nil {
		return orb.Description
	}
	return ""
}
