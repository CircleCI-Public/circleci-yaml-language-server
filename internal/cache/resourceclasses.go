package cache

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"go.lsp.dev/uri"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/memo"
)

// ResourceClasses remembers the self-hosted runner resource classes of each
// organization, for listLifetime, and which organization each file belongs
// to. Every file of an organization shares its classes.
type ResourceClasses struct {
	mutex     sync.Mutex
	orgOfFile map[uri.URI]string

	classes *memo.Memo[[]string]
}

// SetOrgOfFile records the organization, by slug (such as "gh/acme"), whose
// resource classes a file may name, and fetches them, so that they are in hand
// by the time completion asks. An empty slug means the file's is not known.
func (cache *Cache) SetOrgOfFile(client *circleci.V3Client, file uri.URI, orgSlug string) {
	c := &cache.ResourceClassCache
	c.mutex.Lock()
	c.orgOfFile[file] = orgSlug
	c.mutex.Unlock()

	cache.ResourceClassesOfFile(client, file)
}

// ResourceClassesOfFile returns the resource classes of the organization a
// file belongs to, fetching them only when none are remembered. It returns
// none when the organization is not known or the fetch fails.
func (cache *Cache) ResourceClassesOfFile(client *circleci.V3Client, file uri.URI) []string {
	c := &cache.ResourceClassCache
	c.mutex.Lock()
	orgSlug := c.orgOfFile[file]
	c.mutex.Unlock()

	if orgSlug == "" {
		return nil
	}

	classes, err := c.classes.Get(orgSlug, func() ([]string, error) {
		return listResourceClasses(client, orgSlug)
	})
	if err != nil {
		slog.Warn("listing runner resource classes", "org", orgSlug, "err", err)
		return nil
	}

	return classes
}

// listResourceClasses lists an organization's resource classes by its slug.
// An organization CircleCI does not know has none.
func listResourceClasses(client *circleci.V3Client, orgSlug string) ([]string, error) {
	ctx := context.Background()

	orgID, err := circleci.FetchOrgID(ctx, client, orgSlug)
	if errors.Is(err, circleci.ErrNotFound) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	return circleci.ListRunnerResourceClasses(ctx, client, orgID)
}
