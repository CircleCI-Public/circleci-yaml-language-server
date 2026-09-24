package cache

import "github.com/CircleCI-Public/circleci-yaml-language-server/internal/memo"

// Namespaces remembers whether each registry namespace exists on the API
// host, for memo.FoundLifetime or memo.NotFoundLifetime. A self-hosted runner
// resource class names one, and a config commonly names the same namespace for
// many jobs.
type Namespaces struct {
	namespaces *memo.Memo[bool]
}

// Exists reports whether a namespace exists, calling check only when no
// answer is remembered. An error from check is returned but not remembered.
func (c *Namespaces) Exists(name string, check func() (bool, error)) (bool, error) {
	return c.namespaces.Get(name, check)
}
