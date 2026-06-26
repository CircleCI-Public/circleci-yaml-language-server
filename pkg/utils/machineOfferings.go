package utils

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
)

const CurrentLinuxImage = "ubuntu-2404:current"

// Offerings is the executor -> resource-class -> images catalogue returned by the offerings API.
type Offerings struct {
	Linux   map[string][]string `json:"linux"`
	Windows map[string][]string `json:"windows"`
	MacOS   map[string][]string `json:"macos"`
}

// MachinePair is a machine resource class and the images it can run.
type MachinePair struct {
	ResourceClass string
	Images        []string
}

type MachineOfferingsCache struct {
	cacheMutex sync.Mutex
	offerings  *Offerings
	attempted  bool
}

func (c *MachineOfferingsCache) Set(offerings *Offerings) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.offerings = offerings
	c.attempted = true
}

func (c *MachineOfferingsCache) Get() *Offerings {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	return c.offerings
}

// startAttempt returns true only on the first call, so the API is hit at most once.
func (c *MachineOfferingsCache) startAttempt() bool {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	if c.attempted {
		return false
	}
	c.attempted = true
	return true
}

// machineOfferings returns the catalogue, fetching it from the API at most once. It returns
// nil if the catalogue could not be fetched, so callers skip validation instead of reporting
// false errors.
func machineOfferings(lsContext *LsContext, cache *Cache) *Offerings {
	if cache.MachineOfferingsCache.startAttempt() {
		fetchOfferings(lsContext, cache)
	}
	return cache.MachineOfferingsCache.Get()
}

func fetchOfferings(lsContext *LsContext, cache *Cache) {
	url := fmt.Sprintf("%s/api/v3/offerings", lsContext.Api.HostUrl)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", UserAgent)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return
	}

	// The V3 response wraps the catalogue in a data entity: {"data": {"attributes": {...}}}.
	var body struct {
		Data struct {
			Attributes Offerings `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return
	}

	o := body.Data.Attributes
	if len(o.Linux)+len(o.Windows)+len(o.MacOS) > 0 {
		cache.MachineOfferingsCache.Set(&o)
	}
}

// MachinePairs returns the resource-class/images pairs for machine executors (Linux and
// Windows; macOS is handled separately), or nil if the catalogue was not fetched.
func MachinePairs(lsContext *LsContext, cache *Cache) []MachinePair {
	o := machineOfferings(lsContext, cache)
	if o == nil {
		return nil
	}
	pairs := []MachinePair{}
	for _, group := range []map[string][]string{o.Linux, o.Windows} {
		for class, images := range group {
			pairs = append(pairs, MachinePair{ResourceClass: class, Images: images})
		}
	}
	return pairs
}

func MachineImages(lsContext *LsContext, cache *Cache) []string {
	pairs := MachinePairs(lsContext, cache)
	if pairs == nil {
		return nil
	}
	images := map[string]bool{}
	for _, pair := range pairs {
		for _, image := range pair.Images {
			images[image] = true
		}
	}
	return slices.Collect(maps.Keys(images))
}

func MachineResourceClasses(lsContext *LsContext, cache *Cache) []string {
	pairs := MachinePairs(lsContext, cache)
	if pairs == nil {
		return nil
	}
	classes := []string{}
	for _, pair := range pairs {
		classes = append(classes, pair.ResourceClass)
	}
	return classes
}

func XcodeVersions(lsContext *LsContext, cache *Cache) []string {
	o := machineOfferings(lsContext, cache)
	if o == nil {
		return nil
	}
	versions := map[string]bool{}
	for _, images := range o.MacOS {
		for _, image := range images {
			versions[image] = true
		}
	}
	return slices.Collect(maps.Keys(versions))
}

func MacOSResourceClasses(lsContext *LsContext, cache *Cache) []string {
	o := machineOfferings(lsContext, cache)
	if o == nil {
		return nil
	}
	return slices.Collect(maps.Keys(o.MacOS))
}

// DockerResourceClasses returns the Linux classes Docker runs on, plus the Docker-only
// small and medium+, or nil if the catalogue was not fetched.
func DockerResourceClasses(lsContext *LsContext, cache *Cache) []string {
	o := machineOfferings(lsContext, cache)
	if o == nil {
		return nil
	}
	return append([]string{"small", "medium+"}, slices.Collect(maps.Keys(o.Linux))...)
}

func IsSelfHostedRunner(resourceClass string) bool {
	return len(strings.Split(resourceClass, "/")) > 1
}
