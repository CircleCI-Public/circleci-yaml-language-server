package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

const CurrentLinuxImage = "ubuntu-2404:current"

type Offerings struct {
	Linux   map[string][]string `json:"linux"`
	Windows map[string][]string `json:"windows"`
	MacOS   map[string][]string `json:"macos"`
	// Unlike the lists above, Deprecated is keyed by executor, not resource class, and
	// excludes images already present there.
	Deprecated map[string][]string `json:"deprecated"`
}

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

// machineOfferings fetches the catalog once, holding the lock across the fetch so concurrent
// callers wait for the result instead of racing to a nil. Returns nil on failure, so callers
// skip validation rather than flag valid config.
func machineOfferings(lsContext *LsContext, cache *Cache) *Offerings {
	c := &cache.MachineOfferingsCache
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	if !c.attempted {
		c.attempted = true
		c.offerings = fetchOfferings(lsContext)
	}
	return c.offerings
}

func fetchOfferings(lsContext *LsContext) *Offerings {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/api/v3/catalog/offerings", lsContext.Api.HostUrl)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Add("Circle-Token", lsContext.Api.Token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil
	}

	// The V3 response wraps the catalog in a data entity: {"data": {"attributes": {...}}}.
	var body struct {
		Data struct {
			Attributes Offerings `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil
	}

	o := body.Data.Attributes
	if len(o.Linux)+len(o.Windows)+len(o.MacOS) == 0 {
		return nil
	}
	return &o
}

// MachinePairs covers Linux and Windows machine executors; macOS is handled separately.
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

func DeprecatedMachineImages(lsContext *LsContext, cache *Cache) []string {
	o := machineOfferings(lsContext, cache)
	if o == nil {
		return nil
	}
	images := []string{}
	images = append(images, o.Deprecated["linux"]...)
	images = append(images, o.Deprecated["windows"]...)
	return images
}

func DeprecatedXcodeVersions(lsContext *LsContext, cache *Cache) []string {
	o := machineOfferings(lsContext, cache)
	if o == nil {
		return nil
	}
	versions := []string{}
	for _, image := range o.Deprecated["macos"] {
		// API returns "xcode:<version>"; the config field is the bare version.
		versions = append(versions, strings.TrimPrefix(image, "xcode:"))
	}
	return versions
}

func XcodeVersions(lsContext *LsContext, cache *Cache) []string {
	o := machineOfferings(lsContext, cache)
	if o == nil {
		return nil
	}
	versions := map[string]bool{}
	for _, images := range o.MacOS {
		for _, image := range images {
			// API returns "xcode:<version>"; the config field is the bare version.
			versions[strings.TrimPrefix(image, "xcode:")] = true
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

// DockerResourceClasses is the base Linux classes plus the Docker-only small and medium+.
// The machine-only Linux variants (.gen*, .multi, gpu.*) are excluded as Docker can't use them.
func DockerResourceClasses(lsContext *LsContext, cache *Cache) []string {
	o := machineOfferings(lsContext, cache)
	if o == nil {
		return nil
	}
	// small and medium+ are Docker-only and not part of the offerings API, which only
	// returns machine resource classes, so they are added here.
	classes := []string{"small", "medium+"}
	for class := range o.Linux {
		if strings.Contains(class, ".gen") || strings.Contains(class, ".multi") || strings.HasPrefix(class, "gpu.") {
			continue
		}
		classes = append(classes, class)
	}
	return classes
}

func IsSelfHostedRunner(resourceClass string) bool {
	return len(strings.Split(resourceClass, "/")) > 1
}
