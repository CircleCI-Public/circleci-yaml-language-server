package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func GetAllContextEnvVariables(lsContext *LsContext, cache *Cache, organizationId string, contexts []string) []ContextEnvVariable {
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
	var contexts []ContextResponse

	pageToken := ""

	for {
		res, err := getContext(lsContext, orgID, pageToken)
		if err != nil {
			return err
		}

		for _, c := range res.Items {
			contexts = append(contexts, c)
		}

		if res.NextPageToken == nil {
			break
		}

		pageToken = *res.NextPageToken
	}

	for _, c := range contexts {
		cache.ContextCache.SetOrganizationContext(orgID, &Context{
			Id:           c.ID,
			Name:         c.Name,
			envVariables: envVarNames(c.EnvironmentVariables),
		})
	}

	return nil
}

func getContext(lsContext *LsContext, orgID string, nextPageToken string) (*GetAllContextRes, error) {
	url := fmt.Sprintf("%s/api/v2/context?owner-id=%s&include-env-vars=true&page-token=%s", lsContext.Api.HostUrl, orgID, nextPageToken)
	req, err := http.NewRequest(http.MethodGet, url, nil)
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
	defer io.Copy(io.Discard, res.Body)

	var resp GetAllContextRes
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
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
