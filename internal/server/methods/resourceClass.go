package methods

import (
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/projectslug"
)

// SetResourceClassOfFile records the organization whose self-hosted runner
// resource classes a config may name, and fetches them for completion.
func (methods *Methods) SetResourceClassOfFile(params protocol.DidOpenTextDocumentParams) {
	textDocumentUri := params.TextDocument.URI
	orgSlug := projectslug.OrgSlug(projectslug.FromRepo(textDocumentUri.FsPath()))

	methods.Cache.SetOrgOfFile(methods.Settings().V3Client(), textDocumentUri, orgSlug)
}
