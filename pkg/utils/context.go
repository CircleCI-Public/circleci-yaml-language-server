package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ContextRes struct {
	Items         []Context
	NextPageToken string `json:"next_page_token,omitempty"`
}

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

func GetAllContextEnvVariables(token string, cache *Cache, contexts []string) []ContextEnvVariable {
	var contextEnvVariables []ContextEnvVariable
	for _, context := range contexts {
		cachedContext := cache.ContextCache.GetContext(context)
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

func GetAllContext(lsContext *LsContext, ownerSlug string, nextPageToken string, cache *Cache) (bool, error) {
	var nextPageQuery string
	hasBeenUpdated := false

	if nextPageToken != "" {
		nextPageQuery = fmt.Sprintf("&page-token=%s", nextPageToken)
	} else {
		nextPageQuery = ""
	}
	url := fmt.Sprintf("%s/api/v2/context?owner-id=%s%s", lsContext.Api.HostUrl, ownerSlug, nextPageQuery)

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("Circle-Token", lsContext.Api.Token)

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return false, err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return false, err
	}

	var contextRes ContextRes
	err = json.Unmarshal(body, &contextRes)
	if err != nil {
		return false, err
	}

	for _, context := range contextRes.Items {
		if cache.ContextCache.GetContext(context.Name) == nil {
			hasBeenUpdated = true
			cache.ContextCache.SetContext(&context)
		}
	}

	if contextRes.NextPageToken != "" {
		return GetAllContext(lsContext, ownerSlug, contextRes.NextPageToken, cache)
	}

	return hasBeenUpdated, nil
}

type ContextEnvVariables struct {
	Variable  string
	ContextId string `json:"context_id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type ContextEnvVariablesRes struct {
	Items         []ContextEnvVariables
	NextPageToken string `json:"next_page_token,omitempty"`
}

func GetAllContextAllEnvVariables(lsContext *LsContext, cache *Cache) {
	for _, context := range cache.ContextCache.GetAllContext() {
		getContextEnvVariables(lsContext, *context, "", cache)
	}
}

func getContextEnvVariables(lsContext *LsContext, context Context, nextPageToken string, cache *Cache) error {
	var nextPageQuery string
	if nextPageToken != "" {
		nextPageQuery = fmt.Sprintf("?page-token=%s", nextPageToken)
	} else {
		nextPageQuery = ""
	}
	url := fmt.Sprintf("%s/api/v2/context/%s/environment-variable%s", lsContext.Api.HostUrl, context.Id, nextPageQuery)

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("Circle-Token", lsContext.Api.Token)

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return err
	}

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var contextRes ContextEnvVariablesRes
	err = json.Unmarshal(body, &contextRes)
	if err != nil {
		return err
	}

	for _, envVariable := range contextRes.Items {
		cache.ContextCache.AddEnvVariableToContext(context.Name, envVariable.Variable)
	}

	if contextRes.NextPageToken != "" {
		return getContextEnvVariables(lsContext, context, contextRes.NextPageToken, cache)
	}

	return nil
}
