package cache

import (
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
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

// ContextEnvVariables lists the environment variables of the named contexts
// of an organization, as far as they have been cached.
func (c *Cache) ContextEnvVariables(organizationId string, contexts []string) []ContextEnvVariable {
	var contextEnvVariables []ContextEnvVariable
	for _, context := range contexts {
		envVariables, ok := c.ContextCache.envVariablesOf(organizationId, context)
		if !ok {
			continue
		}

		for _, envVariable := range envVariables {
			contextEnvVariables = append(contextEnvVariables, ContextEnvVariable{
				Name:              envVariable,
				AssociatedContext: context,
			})
		}
	}

	return contextEnvVariables
}

// LoadContexts caches the contexts of an organization. Whatever was read before
// a failure is still cached, and the error reports that the rest is missing.
func (c *Cache) LoadContexts(api circleci.Config, orgID string) error {
	contexts, err := circleci.ListContexts(api, orgID, false)

	for _, context := range contexts {
		c.ContextCache.SetOrganizationContext(orgID, &Context{
			Id:           context.ID,
			Name:         context.Name,
			CreatedAt:    context.CreatedAt.String(),
			envVariables: envVarNames(context.EnvironmentVariables),
		})
	}

	return err
}

// LoadContextEnvVariables caches the contexts of an organization along with
// their environment variable names, when the token has permission to read
// them. Used for completion; validation needs only LoadContexts.
func (c *Cache) LoadContextEnvVariables(api circleci.Config, orgID string) error {
	contexts, err := circleci.ListContexts(api, orgID, true)

	for _, context := range contexts {
		existing := c.ContextCache.GetOrganizationContext(orgID, context.Name)
		if existing != nil {
			c.ContextCache.setEnvVariables(existing, envVarNames(context.EnvironmentVariables))
			continue
		}
		c.ContextCache.SetOrganizationContext(orgID, &Context{
			Id:           context.ID,
			Name:         context.Name,
			CreatedAt:    context.CreatedAt.String(),
			envVariables: envVarNames(context.EnvironmentVariables),
		})
	}

	return err
}

func envVarNames(resources []circleci.EnvVar) []string {
	var envVariables []string
	for _, resource := range resources {
		envVariables = append(envVariables, resource.Variable)
	}
	return envVariables
}
