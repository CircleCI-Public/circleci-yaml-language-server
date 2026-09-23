package circleci

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
)

// newV2Client returns a client for the CircleCI V2 REST API on the configured
// host. V2 authenticates with a Circle-Token header rather than a bearer token.
func newV2Client(apiContext Config) *httpcl.Client {
	return client.New(httpcl.Config{
		BaseURL:    apiContext.HostUrl + "/api/v2",
		AuthToken:  apiContext.Token,
		AuthHeader: "Circle-Token",
	})
}
