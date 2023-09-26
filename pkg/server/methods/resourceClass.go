package methods

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

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

func getResourceClassOfOrg(textDocumentUri protocol.URI, context *utils.LsContext) []string {
	projectSlug := utils.GetProjectSlug(textDocumentUri.Filename())
	org := utils.GetProjectOrg(projectSlug)

	if org == "" {
		return []string{}
	}

	hostUrl, err := url.Parse(context.Api.HostUrl)
	if err != nil {
		return []string{}
	}
	hostUrl.Host = "runner." + hostUrl.Host
	url := fmt.Sprintf("%s/api/v3/runner/resource?namespace=%s", hostUrl, org)

	req, _ := http.NewRequest("GET", url, nil)

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
