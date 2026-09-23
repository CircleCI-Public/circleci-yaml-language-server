package circleci

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
)

type EnvVar struct {
	Variable       string    `json:"variable"`
	TruncatedValue string    `json:"truncated_value"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ContextResponse struct {
	Name                 string    `json:"name"`
	ID                   string    `json:"id"`
	CreatedAt            time.Time `json:"created_at"`
	EnvironmentVariables []EnvVar  `json:"environment_variables"`
}

type contextPage struct {
	Items         []ContextResponse `json:"items"`
	NextPageToken *string           `json:"next_page_token"`
}

// ListContexts reads every context of an organization, following the page
// tokens. It reports the contexts it read before any failure, so a caller can
// use a partial answer.
//
// includeEnvVars asks for each context's environment variable names too, which
// needs permission to read them: many users can list contexts but are refused
// with a 403 once the variables are included (private contexts).
func ListContexts(api Config, orgID string, includeEnvVars bool) ([]ContextResponse, error) {
	var contexts []ContextResponse

	pageToken := ""

	for {
		res, err := getContext(api, orgID, pageToken, includeEnvVars)
		if err != nil {
			return contexts, err
		}

		contexts = append(contexts, res.Items...)

		if res.NextPageToken == nil {
			return contexts, nil
		}

		pageToken = *res.NextPageToken
	}
}

func getContext(api Config, orgID string, nextPageToken string, includeEnvVars bool) (*contextPage, error) {
	opts := []func(*httpcl.Request){
		httpcl.QueryParam("owner-id", orgID),
		// The first page is asked for without a page-token at all, rather than
		// with an empty one, so that the request says what it means.
		httpcl.OptionalQueryParam("page-token", nextPageToken),
	}

	if includeEnvVars {
		opts = append(opts, httpcl.QueryParam("include-env-vars", "true"))
	}

	var resp contextPage
	opts = append(opts, httpcl.JSONDecoder(&resp))

	_, err := newV2Client(api).Call(context.Background(), httpcl.NewRequest(http.MethodGet, "/context", opts...))
	if err != nil {
		return nil, fmt.Errorf("list contexts (owner-id=%s): %w", orgID, err)
	}

	return &resp, nil
}
