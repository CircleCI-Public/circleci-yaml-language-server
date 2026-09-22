package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
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
	query := url.Values{}
	query.Set("owner-id", orgID)

	if includeEnvVars {
		// Requires permission to read context environment variables; many users can list
		// contexts but receive HTTP 403 when env vars are included (private contexts).
		query.Set("include-env-vars", "true")
	}

	// The first page is asked for without a page-token at all, rather than with
	// an empty one, so that the request says what it means.
	if nextPageToken != "" {
		query.Set("page-token", nextPageToken)
	}

	requestUrl := fmt.Sprintf("%s/api/v2/context?%s", lsContext.Api.HostUrl, query.Encode())

	req, err := http.NewRequest(http.MethodGet, requestUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Circle-Token", lsContext.Api.Token)
	req.Header.Set("User-Agent", UserAgent)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("list contexts (owner-id=%s): HTTP %d: %s", orgID, res.StatusCode, string(body))
	}

	var resp GetAllContextRes
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, err
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
