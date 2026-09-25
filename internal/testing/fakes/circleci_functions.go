package fakes

// This file serves the functions catalog: the V3 function package and
// version routes that list published functions and carry each version's
// descriptor.

import (
	"net/http"
	"sort"
)

// functionsState is the stored state of the functions catalog.
type functionsState struct {
	packages map[string]FunctionPackage // name -> package
}

// FunctionPackage is a published function.
type FunctionPackage struct {
	ID            string
	Name          string // for example "github.com/circleci-functions/setup-go"
	Description   string
	LatestVersion string
	Versions      []FunctionVersion
}

// FunctionVersion is a published version of a function, with its
// descriptor, the function.yaml as JSON.
type FunctionVersion struct {
	ID         string
	Version    string
	Descriptor map[string]any
}

// AddFunction registers a published function and its versions. The latest
// version is the last one given.
func (f *CircleCI) AddFunction(id, name, description string, versions ...FunctionVersion) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.functions.packages == nil {
		f.functions.packages = map[string]FunctionPackage{}
	}

	latest := ""
	if len(versions) > 0 {
		latest = versions[len(versions)-1].Version
	}

	f.functions.packages[name] = FunctionPackage{
		ID:            id,
		Name:          name,
		Description:   description,
		LatestVersion: latest,
		Versions:      versions,
	}
}

func (f *CircleCI) handleListFunctionPackages(w http.ResponseWriter, r *http.Request) {
	nameFilter := r.URL.Query().Get("filter[name]")

	f.mu.RLock()
	names := make([]string, 0, len(f.functions.packages))
	for name := range f.functions.packages {
		if nameFilter == "" || name == nameFilter {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	entities := make([]any, 0, len(names))
	for _, name := range names {
		function := f.functions.packages[name]
		versions := make([]any, 0, len(function.Versions))
		for _, version := range function.Versions {
			versions = append(versions, map[string]any{
				"id":         version.ID,
				"attributes": map[string]any{"version": version.Version},
			})
		}

		entities = append(entities, map[string]any{
			"id": function.ID,
			"attributes": map[string]any{
				"name":           function.Name,
				"description":    function.Description,
				"latest_version": function.LatestVersion,
			},
			"references": map[string]any{"versions": versions},
		})
	}
	f.mu.RUnlock()

	// The real route answers without a page member.
	writeJSON(w, http.StatusOK, map[string]any{"data": entities})
}

func (f *CircleCI) handleGetFunctionVersion(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	f.mu.RLock()
	defer f.mu.RUnlock()

	for _, function := range f.functions.packages {
		for _, version := range function.Versions {
			if version.ID != id {
				continue
			}

			writeJSON(w, http.StatusOK, map[string]any{
				"data": map[string]any{
					"id": version.ID,
					"attributes": map[string]any{
						"version":    version.Version,
						"descriptor": version.Descriptor,
					},
					"references": map[string]any{"function": map[string]any{"id": function.ID}},
				},
			})
			return
		}
	}

	writeError(w, http.StatusNotFound, "", "function version not found", "")
}
