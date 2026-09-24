package methods

import (
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/projectslug"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/session"
)

func (methods *Methods) SetResourceClassOfFile(params protocol.DidOpenTextDocumentParams) {
	resourceClasses := getResourceClassOfOrg(params.TextDocument.URI, methods.Settings)

	methods.Cache.ResourceClassCache.SetResourceClassForFile(params.TextDocument.URI, &resourceClasses)
}

// getResourceClassOfOrg lists the self-hosted runner resource classes of the
// organization a config belongs to, or none when that cannot be known.
func getResourceClassOfOrg(textDocumentUri uri.URI, lsContext *session.Settings) []string {
	projectSlug := projectslug.FromRepo(textDocumentUri.FsPath())
	org := projectslug.Org(projectSlug)

	if org == "" {
		return []string{}
	}

	resourceClasses, err := circleci.ListRunnerResourceClasses(lsContext.Api, org)
	if err != nil {
		return []string{}
	}

	return resourceClasses
}
