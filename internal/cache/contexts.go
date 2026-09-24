package cache

import (
	"net/http"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/httpcl"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/memo"
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

// Contexts remembers the contexts of each organization, for listLifetime.
type Contexts struct {
	orgs *memo.Memo[*orgContexts]
}

// orgContexts is one listing of an organization's contexts, indexed by each
// name a config may use for one. It is never changed once built, so it is read
// without a lock.
type orgContexts struct {
	byName map[string]*Context
}

// indexContexts indexes contexts by name and, where it is unambiguous, by the
// part of the name after its last "/". The API may return a qualified name
// (org/context) where configs often use the short one for the project's org.
// A short name is never indexed over a context's full name.
func indexContexts(contexts []*Context) *orgContexts {
	byName := make(map[string]*Context, len(contexts))
	for _, ctx := range contexts {
		byName[ctx.Name] = ctx
	}

	shortNames := map[string][]*Context{}
	for _, ctx := range contexts {
		i := strings.LastIndex(ctx.Name, "/")
		if i < 0 || i == len(ctx.Name)-1 {
			continue
		}
		short := ctx.Name[i+1:]
		shortNames[short] = append(shortNames[short], ctx)
	}
	for short, named := range shortNames {
		if _, taken := byName[short]; !taken && len(named) == 1 {
			byName[short] = named[0]
		}
	}

	return &orgContexts{byName: byName}
}

// LoadContexts makes sure the contexts of an organization are remembered,
// listing them only when they are not. A failed listing remembers nothing, and
// its error is returned.
//
// The contexts are listed with their environment variable names, for
// completion. Many users can list contexts but are refused once the variables
// are included (private contexts), so a refusal lists them again without.
func (c *Cache) LoadContexts(api circleci.Config, orgID string) error {
	_, err := c.ContextCache.orgs.Get(orgID, func() (*orgContexts, error) {
		listed, err := circleci.ListContexts(api, orgID, true)
		if httpcl.HasStatusCode(err, http.StatusForbidden) {
			listed, err = circleci.ListContexts(api, orgID, false)
		}
		if err != nil {
			return nil, err
		}

		contexts := make([]*Context, 0, len(listed))
		for _, context := range listed {
			contexts = append(contexts, &Context{
				Id:           context.ID,
				Name:         context.Name,
				CreatedAt:    context.CreatedAt.String(),
				envVariables: envVarNames(context.EnvironmentVariables),
			})
		}

		return indexContexts(contexts), nil
	})

	return err
}

// IsOrganizationContextListLoaded reports whether the contexts of an
// organization are remembered, and so whether a context missing from them can
// be reported as missing.
func (c *Contexts) IsOrganizationContextListLoaded(organizationId string) bool {
	_, ok := c.orgs.Peek(organizationId)
	return ok
}

// ResolveWorkflowContext looks up a context as referenced in config: exact key, then common
// alternates between short vs org-qualified names (API and config do not always agree).
func (c *Contexts) ResolveWorkflowContext(organizationId, organizationSlug, workflowContextName string) *Context {
	org, ok := c.orgs.Peek(organizationId)
	if !ok {
		return nil
	}
	if ctx := org.byName[workflowContextName]; ctx != nil {
		return ctx
	}
	if i := strings.LastIndex(workflowContextName, "/"); i >= 0 {
		short := workflowContextName[i+1:]
		if short != "" {
			if ctx := org.byName[short]; ctx != nil {
				return ctx
			}
		}
	}
	if organizationSlug != "" && !strings.Contains(workflowContextName, "/") {
		if ctx := org.byName[organizationSlug+"/"+workflowContextName]; ctx != nil {
			return ctx
		}
	}
	return nil
}

// ContextEnvVariables lists the environment variables of the named contexts
// of an organization, as far as they are remembered.
func (c *Cache) ContextEnvVariables(organizationId string, contexts []string) []ContextEnvVariable {
	org, ok := c.ContextCache.orgs.Peek(organizationId)
	if !ok {
		return nil
	}

	var contextEnvVariables []ContextEnvVariable
	for _, name := range contexts {
		ctx := org.byName[name]
		if ctx == nil {
			continue
		}

		for _, envVariable := range ctx.envVariables {
			contextEnvVariables = append(contextEnvVariables, ContextEnvVariable{
				Name:              envVariable,
				AssociatedContext: name,
			})
		}
	}

	return contextEnvVariables
}

func envVarNames(resources []circleci.EnvVar) []string {
	var envVariables []string
	for _, resource := range resources {
		envVariables = append(envVariables, resource.Variable)
	}
	return envVariables
}
