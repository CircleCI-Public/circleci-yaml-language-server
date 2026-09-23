package cache

// Namespaces remembers whether each registry namespace exists on the API
// host, for foundLifetime or notFoundLifetime. A self-hosted runner resource class names one, and a config commonly
// names the same namespace for many jobs.
type Namespaces struct {
	namespaces *memo[bool]
}

// Exists reports whether a namespace exists, calling check only when no
// answer is remembered. An error from check is returned but not remembered.
func (c *Namespaces) Exists(name string, check func() (bool, error)) (bool, error) {
	return c.namespaces.get(name, check)
}
