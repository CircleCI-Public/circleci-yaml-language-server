package circleci

import (
	"context"
	"net/url"
	"strings"
)

func IsSelfHostedRunner(resourceClass string) bool {
	return len(strings.Split(resourceClass, "/")) > 1
}

// FetchOrgID looks an organization up by its slug, such as "gh/acme".
//
// Returns ErrNotFound when no organization has that slug.
func FetchOrgID(ctx context.Context, cl *V3Client, slug string) (string, error) {
	query := url.Values{}
	query.Set("filter[slug]", slug)

	orgs, err := GetPaged[struct {
		ID string `json:"id"`
	}](ctx, cl, "orgs", query)
	if err != nil {
		return "", err
	}

	if len(orgs) == 0 {
		return "", ErrNotFound
	}

	return orgs[0].ID, nil
}

// ListRunnerResourceClasses lists the names of an organization's self-hosted
// runner resource classes.
//
// They are listed by organization, not by namespace: the "namespace/" a class
// is named with is an orb namespace the organization has claimed, which need
// not share the organization's name, and one organization can hold classes
// under several namespaces.
func ListRunnerResourceClasses(ctx context.Context, cl *V3Client, orgID string) ([]string, error) {
	query := url.Values{}
	query.Set("filter[org_id]", orgID)

	items, err := GetPaged[struct {
		Attributes struct {
			ResourceClass string `json:"resource_class"`
		} `json:"attributes"`
	}](ctx, cl, "runner/resource-classes", query)
	if err != nil {
		return nil, err
	}

	resourceClasses := make([]string, 0, len(items))
	for _, item := range items {
		resourceClasses = append(resourceClasses, item.Attributes.ResourceClass)
	}

	return resourceClasses, nil
}
