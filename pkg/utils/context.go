package utils

import (
	"strings"
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

type GetAllContextRes struct {
	Organization struct {
		Id       string
		Contexts struct {
			Edges []struct {
				Node struct {
					Groups struct {
						Edges []struct {
							Node struct {
								Name string
							}
						}
					}
					Id        string
					Name      string
					Resources []struct {
						Variable string
					}
				}
			}
			PageInfo struct {
				HasPreviousPage bool
				HasNextPage     bool
			}
			TotalCount int
		}
	}
}

func GetAllContext(lsContext *LsContext, organization string, vcs string, cache *Cache) error {
	cl := NewClient("https://circleci.com", "graphql-unstable", "", false)

	query := `query($organization: String!, $vcsType: VCSType!) {
		organization(vcsType: $vcsType, name: $organization) {
		  id
		  contexts {
			edges {
			  node {
				groups {
				  edges {
					node {
					  name
					}
				  }
				}
				id
				name
				resources {
				  variable
				}
			  }
			}
			pageInfo {
			  hasPreviousPage
			  hasNextPage
			}
			totalCount
		  }
		}
	  }`

	request := NewRequest(query)
	request.Var("organization", organization)
	request.Var("vcsType", strings.ToUpper(vcs))
	request.SetToken(lsContext.Api.Token)
	request.SetUserId(lsContext.Api.userId)

	var Response GetAllContextRes
	err := cl.Run(request, &Response)
	if err != nil {
		return err
	}

	for _, context := range Response.Organization.Contexts.Edges {
		cache.ContextCache.SetOrganizationContext(organization, &Context{
			Id:           context.Node.Id,
			Name:         context.Node.Name,
			envVariables: resourcesToStringArray(context.Node.Resources),
		})
	}

	return nil
}

func resourcesToStringArray(resources []struct {
	Variable string
}) []string {
	var envVariables []string
	for _, resource := range resources {
		envVariables = append(envVariables, resource.Variable)
	}
	return envVariables
}
