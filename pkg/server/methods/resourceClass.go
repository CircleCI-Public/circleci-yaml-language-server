package methods

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"github.com/segmentio/encoding/json"
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
// {"data": ..., "page": ...} envelope — which is why this is a hand-rolled
// request instead of a call through pkg/utils/v3client.go.
//
// The intent is to move onto the standard API. The CLI already reaches this
// same path on the main host rather than on a runner. subdomain, and the newer
// /api/v3/runner/resource-classes endpoint answers with the real V3 envelope,
// filters and all. Until this moves, the host is overridable; see
// utils.ApiContext.RunnerHostUrl.
func getResourceClassOfOrg(textDocumentUri protocol.URI, context *utils.LsContext) []string {
	projectSlug := utils.GetProjectSlug(textDocumentUri.Filename())
	org := utils.GetProjectOrg(projectSlug)

	if org == "" {
		return []string{}
	}

	runnerHost, err := context.Api.RunnerHostUrl()
	if err != nil {
		return []string{}
	}
	url := fmt.Sprintf("%s/api/v3/runner/resource?namespace=%s", runnerHost, org)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []string{}
	}

	req.Header.Add("Circle-Token", context.Api.Token)
	req.Header.Set("User-Agent", utils.UserAgent)

	res, err := http.DefaultClient.Do(req)

	if err != nil || res.StatusCode != 200 {
		return []string{}
	}

	var resourceClassResponse ResourceClassResponse
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return []string{}
	}

	err = json.Unmarshal(body, &resourceClassResponse)
	if err != nil {
		return []string{}
	}

	var resourceClasses []string
	for _, item := range resourceClassResponse.Items {
		resourceClasses = append(resourceClasses, item.ResourceClass)
	}

	return resourceClasses
}
