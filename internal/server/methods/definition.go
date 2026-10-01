package methods

import (
	"context"

	"go.lsp.dev/protocol"

	languageservice "github.com/CircleCI-Public/circleci-yaml-language-server/internal/services"
)

func (methods *Methods) Definition(_ context.Context, params *protocol.DefinitionParams) (protocol.DefinitionResult, error) {
	settings := methods.Settings()
	res, err := languageservice.Definition(*params, methods.Cache, settings)
	if err != nil {
		return nil, err
	}
	// Nothing found has always been answered with null; as a slice, even a
	// nil one would go out as [].
	if res == nil {
		return nil, nil
	}

	// A link says which text it was found from. Without it, a client such as
	// IntelliJ takes a YAML key, a job's name in a workflow for one, to be
	// the declaration itself, and shows its usages instead of going to it.
	if settings.DefinitionLinks {
		links := make(protocol.DefinitionLinkSlice, 0, len(res))
		for _, link := range res {
			links = append(links, link.DefinitionLink())
		}
		return links, nil
	}

	locations := make(protocol.LocationSlice, 0, len(res))
	for _, link := range res {
		locations = append(locations, link.Location())
	}
	return locations, nil
}
