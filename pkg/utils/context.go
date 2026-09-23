package utils

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
)

type Context struct {
	Id           string
	Name         string
	CreatedAt    string `json:"created_at"`
	envVariables []string
}

type ContextEnvVariable struct {
	Name              string
	AssociatedContext string
}

func GetAllContextEnvVariables(cache *Cache, organizationId string, contexts []string) []ContextEnvVariable {
	var contextEnvVariables []ContextEnvVariable
	for _, context := range contexts {
		cachedContext := cache.ContextCache.GetOrganizationContext(organizationId, context)
		if cachedContext == nil {
			continue
		}

		for _, envVariable := range cachedContext.envVariables {
			contextEnvVariables = append(contextEnvVariables, ContextEnvVariable{
				Name:              envVariable,
				AssociatedContext: context,
			})
		}
	}

	return contextEnvVariables
}

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

type GetAllContextRes struct {
	Items         []ContextResponse `json:"items"`
	NextPageToken *string           `json:"next_page_token"`
}

func GetAllContext(lsContext *LsContext, orgID string, cache *Cache) error {
	pageToken := ""

	for {
		res, err := getContext(lsContext, orgID, pageToken, false)
		if err != nil {
			return err
		}

		for _, c := range res.Items {
			cache.ContextCache.SetOrganizationContext(orgID, &Context{
				Id:           c.ID,
				Name:         c.Name,
				CreatedAt:    c.CreatedAt.String(),
				envVariables: envVarNames(c.EnvironmentVariables),
			})
		}

		if res.NextPageToken == nil {
			break
		}

		pageToken = *res.NextPageToken
	}

	return nil
}

// GetAllContextWithEnvVars loads contexts including environment variable names when the token
// has permission. Used for completion; callers should prefer GetAllContext for validation.
func GetAllContextWithEnvVars(lsContext *LsContext, orgID string, cache *Cache) error {
	pageToken := ""

	for {
		res, err := getContext(lsContext, orgID, pageToken, true)
		if err != nil {
			return err
		}

		for _, c := range res.Items {
			existing := cache.ContextCache.GetOrganizationContext(orgID, c.Name)
			if existing != nil {
				existing.envVariables = envVarNames(c.EnvironmentVariables)
				continue
			}
			cache.ContextCache.SetOrganizationContext(orgID, &Context{
				Id:           c.ID,
				Name:         c.Name,
				CreatedAt:    c.CreatedAt.String(),
				envVariables: envVarNames(c.EnvironmentVariables),
			})
		}

		if res.NextPageToken == nil {
			break
		}

		pageToken = *res.NextPageToken
	}

	return nil
}

func getContext(lsContext *LsContext, orgID string, nextPageToken string, includeEnvVars bool) (*GetAllContextRes, error) {
	opts := []func(*httpcl.Request){
		httpcl.QueryParam("owner-id", orgID),
		// The first page is asked for without a page-token at all, rather than
		// with an empty one, so that the request says what it means.
		httpcl.OptionalQueryParam("page-token", nextPageToken),
	}

	if includeEnvVars {
		// Requires permission to read context environment variables; many users can list
		// contexts but receive HTTP 403 when env vars are included (private contexts).
		opts = append(opts, httpcl.QueryParam("include-env-vars", "true"))
	}

	var resp GetAllContextRes
	opts = append(opts, httpcl.JSONDecoder(&resp))

	_, err := newV2Client(lsContext.Api).Call(context.Background(), httpcl.NewRequest(http.MethodGet, "/context", opts...))
	if err != nil {
		return nil, fmt.Errorf("list contexts (owner-id=%s): %w", orgID, err)
	}

	return &resp, nil
}

func envVarNames(resources []EnvVar) []string {
	var envVariables []string
	for _, resource := range resources {
		envVariables = append(envVariables, resource.Variable)
	}
	return envVariables
}
