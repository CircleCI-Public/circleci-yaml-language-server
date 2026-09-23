package hover

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
)

func HoverInOrbs(doc yamlparser.YamlDocument, path []string, cache *cache.Cache) string {
	if len(path) == 0 {
		return commands
	}

	orbName := path[0]
	if len(path) == 1 {
		return orbDefinition(doc, orbName, cache)
	}

	return ""
}

func orbDefinition(doc yamlparser.YamlDocument, orbName string, cache *cache.Cache) string {
	orbInDoc := doc.Orbs[orbName]
	orb, err := doc.GetOrbInfoFromName(orbInDoc.Name, cache)

	if err != nil || orb == nil {
		return ""
	}

	return orb.Description
}
