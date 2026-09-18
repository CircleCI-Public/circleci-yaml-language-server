package fakes

// This file serves the machine catalog: the resource classes and images the
// language server validates and completes executors against.

import "net/http"

// catalogState is the stored machine catalog.
type catalogState struct {
	offerings MachineOfferings
}

// MachineOfferings is the machine catalog GET /api/v3/catalog/offerings
// reports. Linux, Windows and MacOS are keyed by resource class; Deprecated is
// keyed by executor, which is how the real API reports it.
type MachineOfferings struct {
	Linux      map[string][]string
	Windows    map[string][]string
	MacOS      map[string][]string
	Deprecated map[string][]string
}

// SetMachineOfferings registers the machine catalog. Until it is called the
// route answers with an empty catalog, which is the shape a caller has to treat
// as "no offerings" rather than as an error.
func (f *CircleCI) SetMachineOfferings(offerings MachineOfferings) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.catalog.offerings = offerings
}

func (f *CircleCI) handleGetOfferings(w http.ResponseWriter, _ *http.Request) {
	f.mu.RLock()
	offerings := f.catalog.offerings
	f.mu.RUnlock()

	// The V3 response wraps the catalog in a data entity, and reports each
	// group as an object rather than omitting an empty one.
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"attributes": map[string]any{
				"linux":      orEmpty(offerings.Linux),
				"windows":    orEmpty(offerings.Windows),
				"macos":      orEmpty(offerings.MacOS),
				"deprecated": orEmpty(offerings.Deprecated),
			},
		},
	})
}

// orEmpty renders a nil map as {} rather than as null, which is what the real
// API does and what keeps a decoding caller from having to allow for both.
func orEmpty(group map[string][]string) map[string][]string {
	if group == nil {
		return map[string][]string{}
	}

	return group
}
