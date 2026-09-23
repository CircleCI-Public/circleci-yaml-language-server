package fakes

// This file serves the orb registry over GraphQL, which is the only orb API a
// CircleCI Server instance carries. It answers from the same stored state as
// the V3 routes in circleci_orbs.go, so a test seeds data once and can drive
// either transport.

import (
	"encoding/json"
	"net/http"
	"strings"
)

// graphqlState is what the GraphQL view of the registry reports beyond the
// stored orbs themselves.
type graphqlState struct {
	namespaceHasMore bool // when set, a namespace reports another page of orbs
}

// SetNamespaceHasMoreOrbs makes the GraphQL namespace query report a further
// page of orbs, which the query has no way to fetch.
func (f *CircleCI) SetNamespaceHasMoreOrbs() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.graphql.namespaceHasMore = true
}

// handleGraphQL serves the orb and namespace queries the language server sends
// when a host has no V3 orb routes, from the same stored state the V3 handlers
// use.
//
// It does not parse GraphQL. It matches on which root field a query selects,
// which is enough for the fixed set of queries in internal/client/circleci/orbregistry.go, and
// reproduces the behaviour that matters: a missing orb, version or namespace
// comes back as a null member of data rather than as an error.
func (f *CircleCI) handleGraphQL(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"errors": []any{map[string]any{"message": "could not decode request"}},
		})

		return
	}

	variable := func(name string) string {
		value, _ := body.Variables[name].(string)

		return value
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	// orbVersion is checked before orb: "orbVersion(" does not contain "orb(".
	switch {
	case strings.Contains(body.Query, "orbVersion("):
		f.graphQLOrbVersionLocked(w, variable("orbVersionRef"))
	case strings.Contains(body.Query, "registryNamespace("):
		f.graphQLRegistryNamespaceLocked(w, variable("name"), strings.Contains(body.Query, "orbs("))
	case strings.Contains(body.Query, "orb("):
		f.graphQLOrbLocked(w, variable("orbName"))
	default:
		writeJSON(w, http.StatusOK, map[string]any{
			"errors": []any{map[string]any{"message": "unrecognised query"}},
		})
	}
}

func (f *CircleCI) graphQLOrbLocked(w http.ResponseWriter, name string) {
	id, ok := f.orbs.packagesByName[name]
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"orb": nil}})

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{"orb": f.graphQLOrbNodeLocked(f.orbs.packages[id])},
	})
}

func (f *CircleCI) graphQLOrbVersionLocked(w http.ResponseWriter, ref string) {
	version, ok := f.resolveRefLocked(ref)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"orbVersion": nil}})

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"orbVersion": map[string]any{
				"id":      version.ID,
				"version": version.Version,
				"source":  version.Source,
				"orb":     f.graphQLOrbNodeLocked(f.orbs.packages[version.OrbID]),
			},
		},
	})
}

func (f *CircleCI) graphQLRegistryNamespaceLocked(w http.ResponseWriter, name string, withOrbs bool) {
	id, ok := f.orbs.namespacesByName[name]
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"registryNamespace": nil}})

		return
	}

	namespace := map[string]any{"id": id, "name": name}

	if withOrbs {
		edges := []any{}
		for _, orbName := range f.sortedOrbNamesLocked() {
			orb := f.orbs.packages[f.orbs.packagesByName[orbName]]
			if orb.NsID != id {
				continue
			}
			edges = append(edges, map[string]any{
				"cursor": orb.ID,
				"node":   f.graphQLOrbNodeLocked(orb),
			})
		}
		// A namespace claiming another page has to report a totalCount beyond
		// what it served, or the two contradict each other.
		totalCount := len(edges)
		if f.graphql.namespaceHasMore {
			totalCount++
		}
		namespace["orbs"] = map[string]any{
			"totalCount": totalCount,
			"pageInfo":   map[string]any{"hasNextPage": f.graphql.namespaceHasMore},
			"edges":      edges,
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{"registryNamespace": namespace},
	})
}

// graphQLOrbNodeLocked renders an orb with its released versions, newest first
// and excluding development tags, matching what versions(count:) returns.
func (f *CircleCI) graphQLOrbNodeLocked(orb Orb) map[string]any {
	versions := []any{}
	ids := f.orbs.versionsByOrb[orb.ID]
	for i := len(ids) - 1; i >= 0; i-- {
		version := f.orbs.versions[ids[i]]
		if strings.HasPrefix(version.Version, "dev:") {
			continue
		}
		versions = append(versions, map[string]any{"version": version.Version})
	}

	return map[string]any{
		"id":       orb.ID,
		"name":     orb.Name,
		"versions": versions,
	}
}
