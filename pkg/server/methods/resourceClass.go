package methods

import (
	"context"
	"net/http"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

type ResourceClassResponseItem struct {
	Id            string `json:"id"`
	ResourceClass string `json:"resource_class"`
	Description   string `json:"description"`
}

type ResourceClassResponse struct {
	Items []ResourceClassResponseItem `json:"items"`
}

func (methods *Methods) SetResourceClassOfFile(params protocol.DidOpenTextDocumentParams) {
	resourceClasses := getResourceClassOfOrg(params.TextDocument.URI, methods.LsContext)

	methods.Cache.ResourceClassCache.SetResourceClassForFile(params.TextDocument.URI, &resourceClasses)
}

// getResourceClassOfOrg lists the self-hosted runner resource classes of the
// organization a config belongs to.
//
// The /api/v3/runner/... paths are not the CircleCI V3 API. They are the runner
// service's own versioning, addressed here on a runner. subdomain of the API
// host, and they answer with a plain {"items": [...]} rather than the V3
// {"data": ..., "page": ...} envelope — which is why this makes its own
// request instead of a call through pkg/utils/v3client.go.
//
// The intent is to move onto the standard API. The CLI already reaches this
// same path on the main host rather than on a runner. subdomain, and the newer
// /api/v3/runner/resource-classes endpoint answers with the real V3 envelope,
// filters and all. Until this moves, the host is overridable; see
// utils.ApiContext.RunnerHostUrl.
func getResourceClassOfOrg(textDocumentUri protocol.URI, lsContext *utils.LsContext) []string {
	projectSlug := utils.GetProjectSlug(textDocumentUri.Filename())
	org := utils.GetProjectOrg(projectSlug)

	if org == "" {
		return []string{}
	}

	runnerHost, err := lsContext.Api.RunnerHostUrl()
	if err != nil {
		return []string{}
	}
	client := utils.NewHTTPClient(httpcl.Config{
		BaseURL:    runnerHost,
		AuthToken:  lsContext.Api.Token,
		AuthHeader: "Circle-Token",
	})

	var resourceClassResponse ResourceClassResponse
	status, err := client.Call(context.Background(), httpcl.NewRequest(
		http.MethodGet, "/api/v3/runner/resource",
		httpcl.QueryParam("namespace", org),
		httpcl.JSONDecoder(&resourceClassResponse),
	))
	if err != nil || status != http.StatusOK {
		return []string{}
	}

	var resourceClasses []string
	for _, item := range resourceClassResponse.Items {
		resourceClasses = append(resourceClasses, item.ResourceClass)
	}

	return resourceClasses
}
