package fakes

// This file serves organizations: the route a caller holding only an
// organization's slug, such as "gh/acme", reads its id from.

import (
	"net/http"
)

// orgsState is the stored organizations.
type orgsState struct {
	ids map[string]string // slug -> id
}

func newOrgsState() orgsState {
	return orgsState{ids: map[string]string{}}
}

// AddOrg registers an organization by its slug, such as "gh/acme".
func (f *CircleCI) AddOrg(slug, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.orgs.ids[slug] = id
}

// hasOrgID reports whether an organization with this id is registered. The
// caller holds the lock.
func (f *CircleCI) hasOrgID(id string) bool {
	for _, known := range f.orgs.ids {
		if known == id {
			return true
		}
	}

	return false
}

func (f *CircleCI) handleListOrgs(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Query().Get("filter[slug]")

	f.mu.RLock()
	id, known := f.orgs.ids[slug]
	f.mu.RUnlock()

	// An unknown slug is an empty list, not a 404.
	entities := []any{}
	if known {
		entities = append(entities, map[string]any{
			"id":         id,
			"attributes": map[string]any{"name": lastSegment(slug)},
		})
	}

	// The real route answers without a page member.
	writeJSON(w, http.StatusOK, map[string]any{"data": entities})
}
