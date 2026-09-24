package cache

import (
	"log/slog"
	"sync"

	"go.lsp.dev/uri"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/memo"
)

// ResourceClasses remembers the self-hosted runner resource classes of each
// namespace, for listLifetime, and which namespace each file belongs to.
// Every file of an organization shares its classes.
type ResourceClasses struct {
	mutex           sync.Mutex
	namespaceOfFile map[uri.URI]string

	classes *memo.Memo[[]string]
}

// SetNamespaceOfFile records the namespace whose resource classes a file may
// name, and fetches them, so that they are in hand by the time completion
// asks. An empty namespace means the file's is not known.
func (cache *Cache) SetNamespaceOfFile(api circleci.Config, file uri.URI, namespace string) {
	c := &cache.ResourceClassCache
	c.mutex.Lock()
	c.namespaceOfFile[file] = namespace
	c.mutex.Unlock()

	cache.ResourceClassesOfFile(api, file)
}

// ResourceClassesOfFile returns the resource classes of the namespace a file
// belongs to, fetching them only when none are remembered. It returns none
// when the namespace is not known or the fetch fails.
func (cache *Cache) ResourceClassesOfFile(api circleci.Config, file uri.URI) []string {
	c := &cache.ResourceClassCache
	c.mutex.Lock()
	namespace := c.namespaceOfFile[file]
	c.mutex.Unlock()

	if namespace == "" {
		return nil
	}

	classes, err := c.classes.Get(namespace, func() ([]string, error) {
		return circleci.ListRunnerResourceClasses(api, namespace)
	})
	if err != nil {
		slog.Warn("listing runner resource classes", "namespace", namespace, "err", err)
		return nil
	}

	return classes
}
