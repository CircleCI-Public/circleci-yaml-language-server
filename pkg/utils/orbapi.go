package utils

import (
	"context"
	"errors"
	"net/url"
	"sort"
	"strings"

	"golang.org/x/mod/semver"
)

// Orb and namespace metadata from the V3 API. These are the four primitives
// every orb feature in the language server is built from, kept in one place so
// that callers never hand-write a request.
//
// The routes used here answer unauthenticated requests for public orbs, so all
// of this works for users who have not logged in.

// Namespace is an orb registry namespace.
type Namespace struct {
	ID   string
	Name string
}

// OrbPackage is an orb and its released versions.
type OrbPackage struct {
	ID          string
	Name        string
	IsPrivate   bool
	IsListed    bool
	NamespaceID string
	// Versions holds the orb's released versions, sorted newest first. It
	// excludes development tags (for example "dev:alpha"), which the API only
	// exposes through reference resolution.
	Versions []OrbPackageVersion
}

// OrbPackageVersion is one released version of an orb.
type OrbPackageVersion struct {
	ID        string
	Version   string
	CreatedAt string
}

// OrbVersionRef is the result of resolving an orb reference such as
// "circleci/go@1.7", "circleci/go@volatile" or "circleci/go@dev:alpha" to a
// concrete version.
type OrbVersionRef struct {
	ID           string
	Version      string
	CreatedAt    string
	OrbPackageID string
}

// maxPageLimit is the largest page[limit] the V3 API accepts. Larger values are
// rejected with 400 rather than clamped.
const maxPageLimit = "1000"

// OrbPackageName strips the version from an orb reference, turning
// "circleci/go@1.7.1" into "circleci/go". References without a version are
// returned unchanged.
func OrbPackageName(ref string) string {
	name, _, found := strings.Cut(ref, "@")
	if !found {
		return ref
	}

	return name
}

// FetchOrbPackage looks an orb up by its fully qualified "namespace/orb" name.
//
// One request answers three questions at once: whether the orb exists, its id,
// and every version it has published. filter[name] expects the qualified name;
// the bare orb name matches nothing, even alongside filter[namespace_id].
//
// Returns ErrNotFound when no orb matches, which for this route is a 200 with
// an empty data array rather than a 404.
func FetchOrbPackage(ctx context.Context, cl *V3Client, fullName string) (*OrbPackage, error) {
	query := url.Values{}
	query.Set("filter[name]", fullName)

	packages, err := GetPaged[orbPackageData](ctx, cl, "orb/packages", query)
	if err != nil {
		return nil, err
	}

	if len(packages) == 0 {
		return nil, ErrNotFound
	}

	orb := packages[0].toOrbPackage()

	return &orb, nil
}

// ResolveOrbRef resolves an orb reference to a concrete version.
//
// ref is a full reference such as "circleci/go@1.7.1". Exact versions, partial
// versions ("circleci/go@1", "circleci/go@1.7"), "volatile" and development
// tags all resolve, matching what the GraphQL orbVersionRef argument accepted.
//
// orbPackageID is optional. The published spec marks filter[orb_id] as
// required even though the API currently answers without it, so pass the id
// whenever it is already known and leave it empty otherwise.
//
// Returns ErrNotFound when the reference does not resolve.
func ResolveOrbRef(ctx context.Context, cl *V3Client, ref, orbPackageID string) (*OrbVersionRef, error) {
	query := url.Values{}
	query.Set("filter[ref]", ref)
	if orbPackageID != "" {
		query.Set("filter[orb_id]", orbPackageID)
	}

	versions, err := GetPaged[orbVersionData](ctx, cl, "orb/versions", query)
	if err != nil {
		return nil, err
	}

	if len(versions) == 0 {
		return nil, ErrNotFound
	}

	return &OrbVersionRef{
		ID:           versions[0].ID,
		Version:      versions[0].Attributes.Version,
		CreatedAt:    versions[0].Attributes.CreatedAt,
		OrbPackageID: versions[0].References.OrbPackage.ID,
	}, nil
}

// FetchOrbSource returns the YAML source of an orb version, addressed by the
// version's id rather than its reference. The route answers text/plain.
func FetchOrbSource(ctx context.Context, cl *V3Client, orbVersionID string) (string, error) {
	return cl.GetText(ctx, "orb/versions/"+url.PathEscape(orbVersionID)+"/source", nil)
}

// FetchNamespace looks a registry namespace up by name.
//
// Returns ErrNotFound when the namespace does not exist, which for this route
// is a 404. Callers turning that into a diagnostic must not treat other errors
// the same way: a transport failure is not evidence of absence.
func FetchNamespace(ctx context.Context, cl *V3Client, name string) (*Namespace, error) {
	query := url.Values{}
	query.Set("filter[name]", name)

	var data struct {
		ID         string `json:"id"`
		Attributes struct {
			Name string `json:"name"`
		} `json:"attributes"`
	}

	if err := cl.Get(ctx, "namespaces", query, &data); err != nil {
		return nil, err
	}

	if data.ID == "" {
		return nil, ErrNotFound
	}

	return &Namespace{ID: data.ID, Name: data.Attributes.Name}, nil
}

// ListNamespaceOrbs returns every orb in a namespace, following pagination.
//
// filter[namespace_id] takes a namespace uuid, so callers holding only a name
// need FetchNamespace first.
func ListNamespaceOrbs(ctx context.Context, cl *V3Client, namespaceID string) ([]OrbPackage, error) {
	query := url.Values{}
	query.Set("filter[namespace_id]", namespaceID)
	query.Set("page[limit]", maxPageLimit)

	packages, err := GetPaged[orbPackageData](ctx, cl, "orb/packages", query)
	if err != nil {
		return nil, err
	}

	orbs := make([]OrbPackage, 0, len(packages))
	for _, pkg := range packages {
		orbs = append(orbs, pkg.toOrbPackage())
	}

	return orbs, nil
}

// IsNotFound reports whether err means "this does not exist", as opposed to the
// request having failed.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// orbPackageData is the wire shape of an entry in the orb/packages response.
type orbPackageData struct {
	ID         string `json:"id"`
	Attributes struct {
		Name      string `json:"name"`
		IsPrivate bool   `json:"is_private"`
		IsListed  bool   `json:"is_listed"`
	} `json:"attributes"`
	References struct {
		Namespace struct {
			ID string `json:"id"`
		} `json:"namespace"`
		OrbVersions []struct {
			ID         string `json:"id"`
			Attributes struct {
				Version   string `json:"version"`
				CreatedAt string `json:"created_at"`
			} `json:"attributes"`
		} `json:"orb_versions"`
	} `json:"references"`
}

func (data orbPackageData) toOrbPackage() OrbPackage {
	versions := make([]OrbPackageVersion, 0, len(data.References.OrbVersions))
	for _, version := range data.References.OrbVersions {
		versions = append(versions, OrbPackageVersion{
			ID:        version.ID,
			Version:   version.Attributes.Version,
			CreatedAt: version.Attributes.CreatedAt,
		})
	}
	SortOrbVersionsDesc(versions)

	return OrbPackage{
		ID:          data.ID,
		Name:        data.Attributes.Name,
		IsPrivate:   data.Attributes.IsPrivate,
		IsListed:    data.Attributes.IsListed,
		NamespaceID: data.References.Namespace.ID,
		Versions:    versions,
	}
}

// orbVersionData is the wire shape of an entry in the orb/versions response.
// attributes.source is only populated by the single-entity route with
// include=source, never by the collection.
type orbVersionData struct {
	ID         string `json:"id"`
	Attributes struct {
		Version   string `json:"version"`
		CreatedAt string `json:"created_at"`
	} `json:"attributes"`
	References struct {
		OrbPackage struct {
			ID string `json:"id"`
		} `json:"orb_package"`
	} `json:"references"`
}

// SortOrbVersionsDesc sorts versions newest first.
//
// The API is observed to return them in this order already, but no ordering is
// documented, and callers treat the first entry as the latest version. Sorting
// explicitly removes the assumption. Entries that are not valid semver sort
// last, keeping their relative order.
func SortOrbVersionsDesc(versions []OrbPackageVersion) {
	sort.SliceStable(versions, func(i, j int) bool {
		left, right := "v"+versions[i].Version, "v"+versions[j].Version
		leftValid, rightValid := semver.IsValid(left), semver.IsValid(right)

		if leftValid != rightValid {
			return leftValid
		}
		if !leftValid {
			return false
		}

		return semver.Compare(left, right) > 0
	})
}
