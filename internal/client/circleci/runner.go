package circleci

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
)

func IsSelfHostedRunner(resourceClass string) bool {
	return len(strings.Split(resourceClass, "/")) > 1
}

type RunnerResourceClass struct {
	Id            string `json:"id"`
	ResourceClass string `json:"resource_class"`
	Description   string `json:"description"`
}

type runnerResourceClasses struct {
	Items []RunnerResourceClass `json:"items"`
}

// ListRunnerResourceClasses lists the names of the self-hosted runner resource
// classes of a namespace.
//
// The /api/v3/runner/... paths are not the CircleCI V3 API. They are the runner
// service's own versioning, addressed here on a runner. subdomain of the API
// host, and they answer with a plain {"items": [...]} rather than the V3
// {"data": ..., "page": ...} envelope — which is why this makes its own
// request instead of a call through V3Client.
//
// The intent is to move onto the standard API. The CLI already reaches this
// same path on the main host rather than on a runner. subdomain, and the newer
// /api/v3/runner/resource-classes endpoint answers with the real V3 envelope,
// filters and all. Until this moves, the host is overridable; see
// Config.RunnerHostUrl.
func ListRunnerResourceClasses(api Config, namespace string) ([]string, error) {
	runnerHost, err := api.RunnerHostUrl()
	if err != nil {
		return nil, err
	}
	client := client.New(httpcl.Config{
		BaseURL:    runnerHost,
		AuthToken:  api.Token,
		AuthHeader: "Circle-Token",
	})

	var response runnerResourceClasses
	_, err = client.Call(context.Background(), httpcl.NewRequest(
		http.MethodGet, "/api/v3/runner/resource",
		httpcl.QueryParam("namespace", namespace),
		httpcl.JSONDecoder(&response),
	))
	if err != nil {
		return nil, fmt.Errorf("list runner resource classes of %s: %w", namespace, err)
	}

	var resourceClasses []string
	for _, item := range response.Items {
		resourceClasses = append(resourceClasses, item.ResourceClass)
	}

	return resourceClasses, nil
}
