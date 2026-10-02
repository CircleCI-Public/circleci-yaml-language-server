package methods

import (
	"context"

	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/projectslug"
)

// SetResourceClassOfFile records the organization whose self-hosted runner
// resource classes a config may name, and fetches them for completion.
func (methods *Methods) SetResourceClassOfFile(params protocol.DidOpenTextDocumentParams) {
	textDocumentUri := params.TextDocument.URI
	orgSlug := projectslug.OrgSlug(methods.Cache.ProjectSlugOfFile(context.TODO(), textDocumentUri.FsPath()))

	methods.Cache.SetOrgOfFile(context.TODO(), methods.Settings().V3Client(), textDocumentUri, orgSlug)
}
