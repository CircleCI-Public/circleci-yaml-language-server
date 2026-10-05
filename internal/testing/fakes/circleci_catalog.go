package fakes

// This file serves the machine catalog: the resource classes and images the
// language server validates and completes executors against.

import (
	"net/http"
	"slices"
)

// catalogState is the stored machine catalog.
type catalogState struct {
	offerings MachineOfferings
}

// MachineOfferings is the machine catalog GET /api/v3/catalog/resource-classes
// reports. Each group is keyed by resource class, except Deprecated, which is
// keyed by executor.
type MachineOfferings struct {
	Linux   map[string][]string
	Windows map[string][]string
	MacOS   map[string][]string
	// RemoteDocker holds Docker versions rather than images.
	RemoteDocker map[string][]string
	// Docker's lists are empty: a Docker class takes any image.
	Docker map[string][]string
	// Deprecated is listed on each of the executor's classes, as the API
	// lists it, so an executor with no classes has nowhere to list it.
	Deprecated map[string][]string
	// ResourceClasses names and sizes classes, by executor and then by
	// class. A class it leaves out is reported with no name and no size.
	ResourceClasses map[string]map[string]ResourceClass
}

// ResourceClass is a resource class's name and size.
type ResourceClass struct {
	Name  string
	CPU   int
	RAMMB int
}

// SetMachineOfferings registers the machine catalog. Until it is called the
// route answers with an empty catalog, which is the shape a caller has to treat
// as "no offerings" rather than as an error.
func (f *CircleCI) SetMachineOfferings(offerings MachineOfferings) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.catalog.offerings = offerings
}

func (f *CircleCI) handleGetResourceClasses(w http.ResponseWriter, _ *http.Request) {
	f.mu.RLock()
	offerings := f.catalog.offerings
	f.mu.RUnlock()

	// The V3 response lists each executor, with its classes keyed by name.
	// Every executor is listed, even one with no classes, and every list is
	// an array rather than null, as the real API reports them.
	data := []any{}
	for _, group := range []struct {
		executor string
		images   map[string][]string
	}{
		{executor: "linux", images: offerings.Linux},
		{executor: "windows", images: offerings.Windows},
		{executor: "macos", images: offerings.MacOS},
		{executor: "remote_docker", images: offerings.RemoteDocker},
		{executor: "docker", images: offerings.Docker},
	} {
		described := offerings.ResourceClasses[group.executor]
		names := []string{}
		for name := range group.images {
			names = append(names, name)
		}
		for name := range described {
			if !slices.Contains(names, name) {
				names = append(names, name)
			}
		}

		classes := map[string]any{}
		for _, name := range names {
			class := described[name]
			classes[name] = map[string]any{
				"name":              class.Name,
				"cpu":               class.CPU,
				"ram_mb":            class.RAMMB,
				"images":            orEmpty(group.images[name]),
				"deprecated_images": orEmpty(offerings.Deprecated[group.executor]),
			}
		}
		data = append(data, map[string]any{
			"attributes": map[string]any{
				"executor":         group.executor,
				"resource_classes": classes,
			},
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

// orEmpty renders a nil list as [] rather than as null, which is what the real
// API does and what keeps a decoding caller from having to allow for both.
func orEmpty(list []string) []string {
	if list == nil {
		return []string{}
	}

	return list
}
