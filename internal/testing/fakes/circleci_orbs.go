package fakes

// This file serves the orb registry: the V3 namespace, orb package and orb
// version routes the language server resolves orbs through. circleci_graphql.go
// serves the same domain over GraphQL, for hosts that do not carry these routes.
//
// It differs from the circleci-cli fake in three ways, each because this
// repository's code depends on the behavior:
//
//   - filter[namespace_id] is read under its real bracketed name.
//   - Collections paginate with real opaque page[cursor] values, so the
//     cursor-following loop in utils.GetPaged is actually exercised.
//   - An orb package carries every version it has, not just the latest, which
//     is what the real API returns and what orb version resolution relies on.

import (
	"net/http"
	"strings"
)

// orbState is the stored state of the orb registry.
type orbState struct {
	namespaces       map[string]Namespace  // id -> namespace
	namespacesByName map[string]string     // name -> id
	packages         map[string]Orb        // id -> package
	packagesByName   map[string]string     // "ns/name" -> id
	versions         map[string]OrbVersion // id -> version
	versionsByOrb    map[string][]string   // orb id -> version ids, insertion order
	sourceStatus     map[string]int        // version id -> status override for /source
	v3RoutesGone     bool                  // when set, the V3 orb/namespace routes answer 404
}

func newOrbState() orbState {
	return orbState{
		namespaces:       map[string]Namespace{},
		namespacesByName: map[string]string{},
		packages:         map[string]Orb{},
		packagesByName:   map[string]string{},
		versions:         map[string]OrbVersion{},
		versionsByOrb:    map[string][]string{},
		sourceStatus:     map[string]int{},
	}
}

// Namespace is a stored registry namespace.
type Namespace struct {
	ID   string
	Name string
}

// Orb is a stored orb package.
type Orb struct {
	ID        string
	Name      string // fully qualified "namespace/orb"
	NsID      string
	NsName    string
	IsPrivate bool
	IsListed  bool
}

// OrbVersion is a stored orb version.
type OrbVersion struct {
	ID        string
	OrbID     string
	OrbName   string // fully qualified "namespace/orb"
	Version   string
	Source    string
	CreatedAt string
}

// AddNamespace registers a registry namespace.
func (f *CircleCI) AddNamespace(id, name string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.orbs.namespaces[id] = Namespace{ID: id, Name: name}
	f.orbs.namespacesByName[name] = id
}

// AddOrbPackage registers an orb package. orbName is the bare name; the fully
// qualified name is derived from nsName, matching how the API reports it.
func (f *CircleCI) AddOrbPackage(id, nsID, nsName, orbName string, isPrivate, isListed bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	fullName := nsName + "/" + orbName
	f.orbs.packages[id] = Orb{
		ID:        id,
		Name:      fullName,
		NsID:      nsID,
		NsName:    nsName,
		IsPrivate: isPrivate,
		IsListed:  isListed,
	}
	f.orbs.packagesByName[fullName] = id
}

// AddOrbVersion registers a version of an orb package. createdAt may be empty,
// in which case a fixed timestamp is used.
func (f *CircleCI) AddOrbVersion(id, orbID, orbName, version, source, createdAt string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if createdAt == "" {
		createdAt = "2026-01-15T10:30:00.000Z"
	}

	f.orbs.versions[id] = OrbVersion{
		ID:        id,
		OrbID:     orbID,
		OrbName:   orbName,
		Version:   version,
		Source:    source,
		CreatedAt: createdAt,
	}
	f.orbs.versionsByOrb[orbID] = append(f.orbs.versionsByOrb[orbID], id)
}

// SeedGoOrb loads the circleci namespace and a circleci/go orb with a spread of
// versions: several releases across two major versions, plus a development tag,
// which the API resolves by reference but leaves out of a package's version
// list. It is the shared fixture for orb resolution tests.
//
// The version ids are "ver-<version with dots and colons replaced by dashes>",
// so circleci/go@1.7.1 is "ver-1-7-1" and dev:alpha is "ver-dev-alpha".
func (f *CircleCI) SeedGoOrb() {
	f.AddNamespace("ns-circleci", "circleci")
	f.AddOrbPackage("orb-go", "ns-circleci", "circleci", "go", false, true)

	for _, version := range []string{
		"0.1.0",
		"1.7.0",
		"1.7.1",
		"1.7.3",
		"1.12.0",
		"dev:alpha",
		"4.0.0",
	} {
		id := "ver-" + strings.NewReplacer(".", "-", ":", "-").Replace(version)
		f.AddOrbVersion(id, "orb-go", "circleci/go", version, "# source of "+version+"\n", "")
	}
}

// SetSourceStatus makes the /source route for one orb version answer with
// status instead of its YAML.
func (f *CircleCI) SetSourceStatus(orbVersionID string, status int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.orbs.sourceStatus[orbVersionID] = status
}

// DisableV3OrbRoutes makes every V3 orb and namespace route answer 404, the way
// a CircleCI Server instance does. GraphQL keeps working, so this is the switch
// that exercises the fallback.
func (f *CircleCI) DisableV3OrbRoutes() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.orbs.v3RoutesGone = true
}

// isV3OrbRoute reports whether a path is one of the V3 orb or namespace routes
// that CircleCI Server does not serve.
func isV3OrbRoute(path string) bool {
	return path == "/api/v3/namespaces" || strings.HasPrefix(path, "/api/v3/orb/")
}

func (f *CircleCI) handleGetNamespace(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("filter[name]")
	if name == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "Bad Request.", "filter[name] is required")

		return
	}

	f.mu.RLock()
	id, ok := f.orbs.namespacesByName[name]
	f.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, "", "Not Found.", "")

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id":         id,
			"attributes": map[string]any{"name": name},
		},
	})
}

func (f *CircleCI) handleListOrbPackages(w http.ResponseWriter, r *http.Request) {
	nameFilter := r.URL.Query().Get("filter[name]")
	nsFilter := r.URL.Query().Get("filter[namespace_id]")

	f.mu.RLock()
	matched := []Orb{}
	for _, name := range f.sortedOrbNamesLocked() {
		orb := f.orbs.packages[f.orbs.packagesByName[name]]
		if nameFilter != "" && orb.Name != nameFilter {
			continue
		}
		if nsFilter != "" && orb.NsID != nsFilter {
			continue
		}
		matched = append(matched, orb)
	}

	entities := make([]any, 0, len(matched))
	for _, orb := range matched {
		entities = append(entities, f.orbEntityLocked(orb))
	}
	f.mu.RUnlock()

	f.writePage(w, r, "orb/packages", entities)
}

func (f *CircleCI) handleListOrbVersions(w http.ResponseWriter, r *http.Request) {
	ref := r.URL.Query().Get("filter[ref]")
	orbID := r.URL.Query().Get("filter[orb_id]")
	channel := r.URL.Query().Get("filter[channel]")

	f.mu.RLock()
	defer f.mu.RUnlock()

	// filter[ref] resolves a reference the way the real API does: an exact
	// version, a partial version ("1", "1.7"), "volatile", or a dev tag.
	if ref != "" {
		version, ok := f.resolveRefLocked(ref)
		if !ok {
			writeError(w, http.StatusNotFound, "", "Not Found.", "")

			return
		}
		if orbID != "" && version.OrbID != orbID {
			writeError(w, http.StatusNotFound, "", "Not Found.", "")

			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"data": []any{orbVersionEntity(version, false)},
			"page": map[string]any{"next": nil, "prev": nil},
		})

		return
	}

	entities := []any{}
	for _, id := range f.orbs.versionsByOrb[orbID] {
		version := f.orbs.versions[id]
		isDev := strings.HasPrefix(version.Version, "dev:")
		if channel == "stable" && isDev {
			continue
		}
		if channel == "dev" && !isDev {
			continue
		}
		entities = append(entities, orbVersionEntity(version, false))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": entities,
		"page": map[string]any{"next": nil, "prev": nil},
	})
}

func (f *CircleCI) handleGetOrbVersion(w http.ResponseWriter, r *http.Request) {
	f.mu.RLock()
	version, ok := f.orbs.versions[r.PathValue("id")]
	f.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, "", "Not Found.", "")

		return
	}

	// include is single-valued: only the exact value "source" adds the source,
	// and anything else is ignored rather than rejected.
	includeSource := r.URL.Query().Get("include") == "source"

	writeJSON(w, http.StatusOK, map[string]any{"data": orbVersionEntity(version, includeSource)})
}

func (f *CircleCI) handleGetOrbVersionSource(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	f.mu.RLock()
	version, ok := f.orbs.versions[id]
	status := f.orbs.sourceStatus[id]
	f.mu.RUnlock()

	if status != 0 {
		writeError(w, status, "", http.StatusText(status), "")

		return
	}

	if !ok {
		writeError(w, http.StatusNotFound, "", "Not Found.", "")

		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(version.Source))
}

// orbEntityLocked renders an orb package with every version it has, newest
// first, excluding dev tags. Callers must hold the read lock.
func (f *CircleCI) orbEntityLocked(orb Orb) map[string]any {
	versions := []any{}
	ids := f.orbs.versionsByOrb[orb.ID]
	for i := len(ids) - 1; i >= 0; i-- {
		version := f.orbs.versions[ids[i]]
		if strings.HasPrefix(version.Version, "dev:") {
			continue
		}
		versions = append(versions, map[string]any{
			"id": version.ID,
			"attributes": map[string]any{
				"version":    version.Version,
				"created_at": version.CreatedAt,
			},
		})
	}

	references := map[string]any{
		"namespace": map[string]any{"id": orb.NsID},
	}
	if len(versions) > 0 {
		references["orb_versions"] = versions
	}

	return map[string]any{
		"id": orb.ID,
		"attributes": map[string]any{
			"name":                       orb.Name,
			"is_private":                 orb.IsPrivate,
			"is_listed":                  orb.IsListed,
			"last_30_days_build_count":   0,
			"last_30_days_project_count": 0,
			"last_30_days_org_count":     0,
		},
		"references": references,
	}
}

func orbVersionEntity(version OrbVersion, includeSource bool) map[string]any {
	attributes := map[string]any{
		"version":    version.Version,
		"created_at": version.CreatedAt,
	}
	if includeSource {
		attributes["source"] = version.Source
	}

	return map[string]any{
		"id":         version.ID,
		"attributes": attributes,
		"references": map[string]any{
			"orb_package": map[string]any{"id": version.OrbID},
		},
	}
}

// resolveRefLocked resolves "ns/orb@something" the way the API does. Callers
// must hold the read lock.
func (f *CircleCI) resolveRefLocked(ref string) (OrbVersion, bool) {
	name, wanted, found := strings.Cut(ref, "@")
	if !found {
		return OrbVersion{}, false
	}

	orbID, ok := f.orbs.packagesByName[name]
	if !ok {
		return OrbVersion{}, false
	}

	// Newest first, so the first match wins for volatile and partial refs.
	ids := f.orbs.versionsByOrb[orbID]
	candidates := make([]OrbVersion, 0, len(ids))
	for i := len(ids) - 1; i >= 0; i-- {
		candidates = append(candidates, f.orbs.versions[ids[i]])
	}

	for _, candidate := range candidates {
		if candidate.Version == wanted {
			return candidate, true
		}
	}

	if wanted == "volatile" {
		for _, candidate := range candidates {
			if !strings.HasPrefix(candidate.Version, "dev:") {
				return candidate, true
			}
		}

		return OrbVersion{}, false
	}

	// A partial version matches the newest release under that prefix.
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate.Version, wanted+".") {
			return candidate, true
		}
	}

	return OrbVersion{}, false
}

func (f *CircleCI) sortedOrbNamesLocked() []string {
	names := make([]string, 0, len(f.orbs.packagesByName))
	for name := range f.orbs.packagesByName {
		names = append(names, name)
	}

	// Insertion order is not meaningful for a map, and the real API returns a
	// stable ordering, so sort to keep pagination deterministic.
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && names[j] < names[j-1]; j-- {
			names[j], names[j-1] = names[j-1], names[j]
		}
	}

	return names
}
