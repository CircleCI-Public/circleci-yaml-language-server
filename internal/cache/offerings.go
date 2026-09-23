package cache

import (
	"sync"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
)

type MachineOfferings struct {
	cacheMutex sync.Mutex
	offerings  *circleci.Offerings
	attempted  bool
}

func (c *MachineOfferings) Set(offerings *circleci.Offerings) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.offerings = offerings
	c.attempted = true
}

// Offerings fetches the machine catalog once, holding the lock across the fetch so concurrent
// callers wait for the result instead of racing to a nil. Returns nil on failure, so callers
// skip validation rather than flag valid config; every view of *Offerings answers nil for a
// nil catalog.
func (cache *Cache) Offerings(api circleci.Config) *circleci.Offerings {
	c := &cache.MachineOfferingsCache
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	if !c.attempted {
		c.attempted = true
		c.offerings = circleci.FetchOfferings(api)
	}
	return c.offerings
}
