package hover

import (
	"context"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/cache"
	yamlparser "github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
)

func HoverInOrbs(ctx context.Context, doc yamlparser.YamlDocument, path []string, cache *cache.Cache) string {
	if len(path) == 0 {
		return commands
	}

	orbName := path[0]
	if len(path) == 1 {
		return orbDefinition(ctx, doc, orbName, cache)
	}

	return ""
}

func orbDefinition(ctx context.Context, doc yamlparser.YamlDocument, orbName string, cache *cache.Cache) string {
	orbInDoc := doc.Orbs[orbName]
	orb, err := doc.GetOrbInfoFromName(ctx, orbInDoc.Name, cache)

	if err != nil || orb == nil {
		return ""
	}

	return orb.Description
}
