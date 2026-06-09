package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

const CurrentLinuxImage = "ubuntu-2404:current"

// Offerings is the executor -> resource-class -> images catalogue from the offerings API,
// plus a flat list of deprecated images to warn about.
type Offerings struct {
	Linux      map[string][]string
	Windows    map[string][]string
	MacOS      map[string][]string
	Deprecated []string
}

type MachinePair struct {
	Images          []string
	ResourceClasses []string
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

// startAttempt returns true only the first time, so we call the API at most once.
func (c *MachineOfferingsCache) startAttempt() bool {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	if c.attempted {
		return false
	}
	c.attempted = true
	return true
}

// FetchMachineOfferings gets the offerings from the API and caches them. On any error
// the cache stays empty, so callers skip validation.
func FetchMachineOfferings(lsContext *LsContext, cache *Cache) {
	if !cache.MachineOfferingsCache.startAttempt() {
		return
	}

	url := fmt.Sprintf("%s/api/v3/machine/offerings", lsContext.Api.HostUrl)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return
	}
	req.Header.Add("Circle-Token", lsContext.Api.Token)
	req.Header.Set("User-Agent", UserAgent)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return
	}

	// V3 wraps the catalogue in a single data entity: {"data": {"attributes": {...}}}.
	var envelope struct {
		Data struct {
			Attributes struct {
				Linux      map[string][]string `json:"linux"`
				Windows    map[string][]string `json:"windows"`
				MacOS      map[string][]string `json:"macos"`
				Deprecated []string            `json:"deprecated"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		return
	}

	a := envelope.Data.Attributes
	if len(a.Linux)+len(a.Windows)+len(a.MacOS) > 0 {
		cache.MachineOfferingsCache.Set(&Offerings{
			Linux:      a.Linux,
			Windows:    a.Windows,
			MacOS:      a.MacOS,
			Deprecated: a.Deprecated,
		})
	}
}

func machineOfferings(lsContext *LsContext, cache *Cache) *Offerings {
	FetchMachineOfferings(lsContext, cache)
	return cache.MachineOfferingsCache.Get()
}

// MachinePairs returns the image/resource-class pairs for machine executors (Linux and
// Windows; macOS is checked on its own), or nil if the offerings weren't fetched.
func MachinePairs(lsContext *LsContext, cache *Cache) []MachinePair {
	o := machineOfferings(lsContext, cache)
	if o == nil {
		return nil
	}

	pairs := []MachinePair{}
	for _, group := range []map[string][]string{o.Linux, o.Windows} {
		for class, images := range group {
			// Deprecated images stay valid.
			pairs = append(pairs, MachinePair{
				Images:          append(append([]string{}, images...), o.Deprecated...),
				ResourceClasses: []string{class},
			})
		}
	}
	return pairs
}

func MachineImages(lsContext *LsContext, cache *Cache) []string {
	return dedupe(MachinePairs(lsContext, cache), func(p MachinePair) []string { return p.Images })
}

func MachineResourceClasses(lsContext *LsContext, cache *Cache) []string {
	return dedupe(MachinePairs(lsContext, cache), func(p MachinePair) []string { return p.ResourceClasses })
}

// XcodeVersions returns the macOS images (Xcode versions), including deprecated ones, or
// nil if there are none.
func XcodeVersions(lsContext *LsContext, cache *Cache) []string {
	o := machineOfferings(lsContext, cache)
	if o == nil {
		return nil
	}
	set := map[string]bool{}
	for _, images := range o.MacOS {
		for _, img := range images {
			set[img] = true
		}
	}
	for _, img := range o.Deprecated {
		set[img] = true
	}
	if len(set) == 0 {
		return nil
	}
	return keys(set)
}

func MacOSResourceClasses(lsContext *LsContext, cache *Cache) []string {
	o := machineOfferings(lsContext, cache)
	if o == nil {
		return nil
	}
	classes := []string{}
	for class := range o.MacOS {
		classes = append(classes, class)
	}
	return classes
}

// DockerResourceClasses returns the Linux classes Docker runs on plus the Docker-only
// small and medium+, or nil if the offerings weren't fetched.
func DockerResourceClasses(lsContext *LsContext, cache *Cache) []string {
	o := machineOfferings(lsContext, cache)
	if o == nil {
		return nil
	}
	set := map[string]bool{"small": true, "medium+": true}
	for class := range o.Linux {
		set[class] = true
	}
	return keys(set)
}

func dedupe(pairs []MachinePair, pick func(MachinePair) []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, p := range pairs {
		for _, v := range pick(p) {
			if !seen[v] {
				seen[v] = true
				out = append(out, v)
			}
		}
	}
	return out
}

func keys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}

func IsSelfHostedRunner(resourceClass string) bool {
	return len(strings.Split(resourceClass, "/")) > 1
}
