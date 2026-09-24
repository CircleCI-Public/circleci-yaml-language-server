package cache

import (
	"time"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/memo"
)

// MachineOfferings remembers the machine catalog of the API host.
type MachineOfferings struct {
	catalog *memo.Memo[*circleci.Offerings]
}

// catalogKey is the one key the catalog is kept under: there is one catalog
// per host, and the cache is cleared when the host changes.
const catalogKey = "catalog"

// offeringsLifetime keeps a catalog for memo.FoundLifetime, and the lack of
// one only for memo.NotFoundLifetime.
//
// No catalog is usually a failed fetch, and it is remembered anyway, unlike
// any other failure: validation asks for the catalog for every executor, and
// each would otherwise wait out a request to a host that is down.
func offeringsLifetime(offerings *circleci.Offerings) time.Duration {
	return memo.Existence(offerings != nil)
}

func (c *MachineOfferings) Set(offerings *circleci.Offerings) {
	c.catalog.Put(catalogKey, offerings)
}

// Offerings returns the machine catalog, fetching it only when none is
// remembered; concurrent callers share one fetch. Returns nil on failure, so
// callers skip validation rather than flag valid config; every view of
// *Offerings answers nil for a nil catalog.
func (cache *Cache) Offerings(api circleci.Config) *circleci.Offerings {
	offerings, _ := cache.MachineOfferingsCache.catalog.Get(catalogKey, func() (*circleci.Offerings, error) {
		return circleci.FetchOfferings(api), nil
	})
	return offerings
}
