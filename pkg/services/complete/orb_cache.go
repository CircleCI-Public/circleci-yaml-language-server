package complete

import (
	"context"
	"fmt"
	"sync"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

// OrbCache memoises the orb metadata that autocomplete needs, so that typing
// in an orb reference does not re-query the registry on every keystroke.
type OrbCache struct {
	mutex sync.Mutex
	// registryOrbs maps a namespace name to the orbs it publishes.
	registryOrbs map[string][]OrbData
	orbData      map[string]*OrbData
}

// OrbData is an orb and the versions it has published, newest first.
type OrbData struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Versions []OrbVersion `json:"versions"`
}

// OrbVersion is one published version of an orb.
type OrbVersion struct {
	Version string `json:"version"`
}

// NewOrbCache returns an empty cache.
func NewOrbCache() *OrbCache {
	return &OrbCache{
		registryOrbs: make(map[string][]OrbData),
		orbData:      make(map[string]*OrbData),
	}
}

// GetOrbsOfRegistry returns every orb published in a namespace.
func (cache *OrbCache) GetOrbsOfRegistry(registry, hostUrl, token, userId string) ([]OrbData, error) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	if cached, ok := cache.registryOrbs[registry]; ok {
		return cached, nil
	}

	orbRegistry := utils.NewOrbRegistry(hostUrl, token, userId, false)

	packages, err := orbRegistry.ListNamespaceOrbs(context.Background(), registry)
	if err != nil {
		if utils.IsNotFound(err) {
			return nil, fmt.Errorf("no namespace named %s", registry)
		}

		return nil, err
	}

	orbs := make([]OrbData, 0, len(packages))
	for _, pkg := range packages {
		orbs = append(orbs, toOrbData(pkg))
	}

	cache.registryOrbs[registry] = orbs
	for i := range orbs {
		cache.orbData[orbs[i].Name] = &orbs[i]
	}

	return orbs, nil
}

// GetVersionsOfOrb returns an orb and its versions, given the orb's fully
// qualified "namespace/orb" name.
func (cache *OrbCache) GetVersionsOfOrb(orbName, hostUrl, token, userId string) (*OrbData, error) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	if cached, ok := cache.orbData[orbName]; ok {
		return cached, nil
	}

	orbRegistry := utils.NewOrbRegistry(hostUrl, token, userId, false)

	pkg, err := orbRegistry.FetchOrb(context.Background(), orbName)
	if err != nil {
		if utils.IsNotFound(err) {
			return nil, fmt.Errorf("no orb named %s", orbName)
		}

		return nil, err
	}

	orb := toOrbData(*pkg)
	cache.orbData[orbName] = &orb

	return &orb, nil
}

func toOrbData(pkg utils.OrbPackage) OrbData {
	versions := make([]OrbVersion, 0, len(pkg.Versions))
	for _, version := range pkg.Versions {
		versions = append(versions, OrbVersion{Version: version.Version})
	}

	return OrbData{
		ID:       pkg.ID,
		Name:     pkg.Name,
		Versions: versions,
	}
}
